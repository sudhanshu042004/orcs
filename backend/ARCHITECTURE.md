# ORCS Backend Architecture

ORCS is a self-hosted deploy platform: a user logs in with GitHub, picks a repository and a
project type, and ORCS clones, builds and serves it. A build that produces a directory of
files is published to object storage; one that produces a long-running app keeps its
container and gets proxied to.

Everything runs as **one Go binary** (`cmd/orcs`) on port `3000`. That single process is
the API server, the build worker pool and the public site server at the same time — there is
no separate worker service. Alongside it `docker-compose.yml` runs Postgres, MinIO (S3) and
the Vite frontend.

---

## 1. Modules

15 Go packages. Nothing imports upward, and the dependency graph is acyclic.

| # | Package | Role |
|---|---------|------|
| 1 | `cmd/orcs` | Entrypoint. Builds the Gin router, orders the middleware, starts the worker pool. |
| 2 | `database` | Postgres connection (`database.DB`, a package-level `*sql.DB`) + migration runner. |
| 3 | `types` | Shared DTOs: `User`, `Deployment`, `JwtPayload`. Imports nothing. |
| 4 | `pkg/config` | GitHub OAuth config, CORS middleware, JWT create/verify/cookie, OAuth-state + safe-redirect handling. |
| 5 | `internal/middleware` | `AuthRequired()` — reads the `orcsAuth` cookie, verifies the JWT, puts `id`/`email` on the Gin context. |
| 6 | `internal/auth` | `GET /login`, `GET /auth/callback`. GitHub OAuth dance, find-or-create user, set cookie, redirect. |
| 7 | `internal/user` | `GET /api/user`. |
| 8 | `internal/repository` | All SQL. `user.go` (2 queries) and `deployment.go` (deployments + `deployment_containers`). No HTTP, no docker. |
| 9 | `internal/github-repo` | The deployment HTTP layer: list repos/project types, create/list/delete deployments, NDJSON log stream. |
| 10 | `internal/stack` | Static table of supported project types (react, node, go, rust): image, kind, default commands, output dirs, port. The single source of truth for "how is this built". |
| 11 | `internal/queue` | In-process buffered-channel job queue (cap 128) + worker pool. Generic: takes a `Handler`, knows nothing about builds. |
| 12 | `internal/worker` | The build pipeline. Consumes queue jobs, drives `container`, publishes via `s3`, writes status back via `repository`. |
| 13 | `internal/container` | Thin wrapper over the `docker` CLI (`exec.Command`) + the `buildScript` that runs inside every build container. |
| 14 | `internal/s3` | AWS SDK v2 client (works against MinIO). Tar-stream upload, object read, prefix delete. |
| 15 | `internal/site` | Gin middleware that serves deployed sites on `<id>.<domain>` — from object storage, or by reverse-proxying a running app. |

### Non-Go pieces

- `database/migrations/*.sql` — 6 migrations, applied two ways: mounted into Postgres'
  `docker-entrypoint-initdb.d` **and** re-run by `runMigrations()` at boot (idempotent-ish:
  `already exists` errors are swallowed).
- `uploads/logs/<deployment-id>.log` — the only thing a build writes to the host filesystem.

---

## 2. Dependency graph

```
                              cmd/orcs
        ┌──────────┬──────────┬───┴────┬──────────┬──────────┐
        ▼          ▼          ▼        ▼          ▼          ▼
     database    auth    middleware   user   github-repo   worker ──┐
                   │          │        │          │                 │
                   │          │        │          ├── queue ◄───────┤
                   │          │        │          ├── stack  ◄──────┤
                   │          │        │          ├── container ◄───┤
                   │          │        │          ├── s3     ◄──────┤
                   │          │        │          └── site   ◄──────┘
                   │          │        │                 │
                   │          │        │                 ├── repository
                   │          │        │                 ├── container
                   ▼          ▼        ▼                 ├── s3
              pkg/config  pkg/config  repository         └── stack
                   │                      │
                   ▼                      ├── database
                 types                    └── types
```

Worth noting:

- **`repository` is the only package that touches SQL.** Handlers and the worker both go
  through it; neither builds a query.
- **`stack` is a leaf.** The HTTP layer (validating a deploy request), the worker (running
  the build) and `site` (deciding how to serve it) all read the same table, so they cannot
  disagree about how a project is built.
- **`queue` is a leaf too** — it holds a `Handler` func, so it has no idea `worker` exists.
  `worker.Start()` injects `worker.RunBuild` into it. That is what keeps queue→worker from
  becoming a cycle.
- `github-repo` → `worker` is only for the `StatusDeployed`/`StatusFailed` string constants
  used by the log stream to decide when to stop following.
- `site` is imported by `worker` (for `site.URL()`, `site.RuntimeHost()`) and by
  `github-repo` (for `site.Forget()` on delete) — it is both a request handler and a library.

---

## 3. Request routing order

Middleware order in `cmd/orcs/main.go` matters a lot:

```
request
  │
  ├─ site.Middleware()            ← FIRST. If Host is "<digits>.<something>", this
  │                                 serves the deployed site and aborts the chain.
  ├─ config.CorsMiddleware()
  │
  ├─ GET /health                  (open)
  ├─ GET /login, /auth/callback   (open — auth group)
  │
  └─ middleware.AuthRequired()    ← everything below needs the orcsAuth cookie
       ├─ GET    /api/user
       ├─ GET    /repos
       ├─ GET    /stacks
       ├─ GET    /deployments
       ├─ POST   /deployments
       ├─ DELETE /deployments/:id
       └─ GET    /deployments/:id/stream
```

`site.Middleware()` runs ahead of the API on purpose: deployed sites live on their own hosts
(`20.localhost:3000`), so a site request must never fall through to an API route. The host
parser (`deploymentIdFromHost`) only accepts a **leading all-digit label**, and explicitly
rejects IP literals so `127.0.0.1:3000` isn't read as "deployment 127".

---

## 4. Flows

### 4.1 Login

```
browser  GET /login?callback=/dashboard/deploy
   │
   │  auth.GithubLogin
   │    config.SafeRedirectPath()  — drop absolute / "//evil.com" paths (open-redirect guard)
   │    config.EncodeState()       — "randomstate:<base64 path>"
   └─ 303 → github.com/login/oauth/authorize
        │
        └─ GET /auth/callback?code=…&state=…
             config.DecodeState()           — verify nonce, recover the path
             oauth2.Exchange(code)          — → access token
             GET api.github.com/user        — → login, email, avatar, repos_url
             repository.FindUser(email)
               ├─ found     → config.SetCookie(existing.Id, …)
               └─ not found → repository.CreateUser(…) → config.SetCookie(new.Id, …)
                                 │
                                 ├─ CreateToken  — HS256 JWT, 7-day exp, {id, email}
                                 ├─ Set-Cookie   orcsAuth (HttpOnly, 4000s, path=/)
                                 └─ 302 → FRONTEND_ROUTE + safe path
```

Note the cookie's `Max-Age` (4000s ≈ 67min) is far shorter than the JWT's 7-day expiry, so
the cookie is what actually ends the session.

### 4.2 Authenticated API call

```
request → middleware.AuthRequired
            c.Cookie("orcsAuth") → config.VerifyToken (HS256, JWT_SECRET)
            c.Set("email"), c.Set("id")
          handler reads c.Get("id") → repository.<query> → JSON
```

Every deployment query is scoped by `user_id`, so ownership is enforced in SQL
(`GetDeployment(id, userId)`, `DeleteDeployment(id, userId)`) rather than in the handler.
The one deliberate exception is `GetPublicDeployment(id)`, used by `site` — a deployed site
is served to anyone with the URL, so it cannot ask who is asking.

### 4.3 Deploy (the main flow)

**Phase A — the request (synchronous, fast):**

```
POST /deployments  {name, stack, repo_url, install_cmd, build_cmd, run_cmd}
  │
  github-repo.CreateDeployment
    stack.Get(req.Stack)              — unknown / !Enabled → 400
    if stack.Locked:                  — React only; its commands replace whatever was sent.
      use the stack's own commands      Node/Go/Rust are configurable, so the user's
                                        install/build/run commands are used as submitted.
    url.Parse(repo_url)               — must be http(s) with a host
    install_cmd / build_cmd empty     — → 400
    Kind == dynamic && run_cmd empty  — → 400 ("<Label> projects need a run command")
    repository.CreateDeployment(…)    — INSERT status='queued', url=''  → depId
    queue.Default().Enqueue(job)      — non-blocking; full queue → mark 'failed' + 503
  │
  └─ 200 {deployment_id, status: "queued"}
```

`GET /stacks` serves the `stack` table straight to the deploy form, so each project type
arrives with its `locked` flag and its own commands. The form pre-fills those commands on
every project-type change: for React they are fixed and the inputs are disabled, for
Node/Go/Rust they are a starting point the user can edit before deploying.

**Phase B — the build (asynchronous, on a worker goroutine):**

`buildWorkers = 2`, so two builds run concurrently.

```
queue.work(i) → worker.RunBuild(job)
  │
  buildLog(depId)                       → uploads/logs/<id>.log (O_TRUNC, syncWriter)
  repository.UpdateDeploymentStatus(id, "pending", "")
  container.CreateBuildContainer(spec)  → docker create
  repository.SaveDeploymentContainer(depId, containerId)
  container.StartContainer(containerId) → docker start
  │
  ├─ Kind == static (react) ────────────────────────────────────────┐
  │    docker logs -f  → copied into the log until the container     │
  │                      exits (the build command is the whole job)  │
  │    container.GetContainerExitCode  — non-zero → fail()           │
  │    for dir in stack.OutputDirs (dist, build):                    │
  │      container.CopyTarFromContainer("/app/"+dir)  — docker cp -  │
  │      s3.UploadTarStream(stream, "deployments/<id>", strip=1)     │
  │      first dir that yields >0 files wins                         │
  │    0 files anywhere → fail()                                     │
  │    container.RemoveContainer + DeleteDeploymentContainer         │
  │    UpdateDeploymentStatus("deployed", site.URL(depId))           │
  │                                                                  │
  └─ Kind == dynamic (node/go/rust) → runDynamic() ─────────────────┤
       container.StreamLogsTo(id, log)  — background follow, no wait │
       awaitListening(id, stack.Port)   — up to 2 min:               │
         ├ !IsRunning → read exit code → fail ("exited before …")    │
         ├ container.HostPort(id, 3000) — docker port                │
         └ real HTTP GET to RUNTIME_HOST:hostPort/ — a TCP connect   │
           would succeed too early, since docker publishes the port  │
           the moment the container starts                          │
       repository.SetDeploymentHostPort(depId, hostPort)             │
       container.KeepAlive(id)          — docker update --restart    │
                                          unless-stopped, only AFTER │
                                          the app is known to work   │
       UpdateDeploymentStatus("deployed", site.URL(depId))           │
       log stays OPEN — the app keeps writing to it while it runs ───┘
```

Any failure path goes through the closure `fail()`: log the line, remove the container, drop
the `deployment_containers` row, set status `failed`.

The build itself happens **entirely inside the container** — `buildScript` in
`container/client.go` does the `git clone`, install and build. The repo URL and commands
arrive as **environment variables**, never interpolated into the shell command line. The
script also writes `/app/.orcs-ready`, so a restarted container skips clone+build and goes
straight to `exec`ing the run command.

`worker.RunBuild` re-reads `stack.Locked` rather than trusting the row: a locked project type
is rebuilt with its own commands even if the stored commands say otherwise. Everything else
builds with exactly what the user configured.

**Phase C — status lifecycle:**

```
queued ──(worker picks up)──▶ pending ──▶ deployed
   │                             │
   └─────────────────────────────┴──────▶ failed
```

On process restart, `worker.Start()` calls `repository.GetQueuedDeployments()` and
re-enqueues everything still in `queued`. Postgres is the durable record; the channel is not.
Builds that were `pending` when the process died are **not** recovered — they stay stuck
`pending` (migration 000005 swept the older `building`/`draft` rows to `failed`).

### 4.4 Live build log stream

```
GET /deployments/:id/stream
  │
  repository.GetDeployment(id, userId)      — ownership check
  headers: application/x-ndjson, no-cache, X-Accel-Buffering: no
  │
  write {"type":"status", status, url, name, repo_url}
  loop every 500ms:
    logTail.next()  → tail uploads/logs/<id>.log from a byte offset,
                       holding back a partial line until its \n arrives;
                       a shrinking file (rebuild truncated it) rewinds to 0
    → write {"type":"log","text":…} per line
    re-read the row; status changed → drain the log FIRST, then emit the new status
    status in {deployed, failed} → done
```

This deliberately tails the **file**, not docker. The file exists from the moment the worker
picks the job up, so there is no window between "job dequeued" and "container exists" in
which log lines are lost, and a finished build replays in full for a client that connects late.

### 4.5 Serving a deployed site

```
GET http://20.localhost:3000/assets/index-abc.js
  │
  site.Middleware
    deploymentIdFromHost("20.localhost:3000") → 20
    c.Abort()                        — API routes never see this
    lookup(20)                       — 5s TTL cache; one page load asks for every asset,
                                       so the DB round trip is shared
      └─ resolve(20)
           repository.GetPublicDeployment(20)
           stack.Get(dep.Stack)
           ├─ Kind static  → target{}            (read back from object storage)
           └─ Kind dynamic → container.HostPort(live)  — the LIVE mapping wins over the
                             ?? GetDeploymentHostPort   recorded one, because docker hands
                             0 → "this app is not running"  out a new port on restart
                             → httputil.NewSingleHostReverseProxy
  │
  ├─ proxied → proxy.ServeHTTP            (the app does its own routing; ErrorHandler → 502)
  └─ stored  → GET/HEAD only, else 405
       serve()
         objectKey(20, path) → "deployments/20/assets/index-abc.js"
                               path.Clean inside the prefix — "/../../etc/passwd" can't escape
         s3.GetObject(key)
           ErrNotFound + path has an extension → 404 (a real missing file)
           ErrNotFound + no extension          → retry index.html (SPA client-side route)
         Cache-Control: .html → no-cache; everything else → immutable, 1 year
         ETag passed through
```

`site.Forget(depId)` is called on delete so a deleted deployment isn't served from a stale
cache entry for up to 5 seconds.

S3 objects stay **private**: the site handler holds the credentials and reads them back
server-side, rather than making the bucket public.

### 4.6 Delete

```
DELETE /deployments/:id
  repository.GetDeployment(id, userId)     — ownership
  site.Forget(id)                          — stop proxying to a port about to die
  repository.GetDeploymentContainer → container.RemoveContainer     (warn-only)
  rm -rf uploads/cloned/<userId>/<name>    — legacy, pre-container builds (warn-only)
  rm uploads/logs/<id>.log                                          (warn-only)
  s3.DeletePrefix("deployments/<id>")      — paginated ListObjectsV2 + DeleteObjects
  repository.DeleteDeployment(id, userId)  — deployment_containers cascades
```

Steps 2–5 are best-effort; only the final row delete can fail the request.

---

## 5. Data model

```
users
  id, username, name, email UNIQUE, avatar, repo_url
    │ ON DELETE CASCADE
    ▼
deployments
  id, user_id, name
  status      CHECK IN ('queued','pending','deployed','failed')
  url         — "http://<id>.<base domain>", filled in when deployed
  repo_url
  stack       — FK-in-spirit to internal/stack keys (react|node|go|rust)
  install_cmd, build_cmd, run_cmd
  created_at
    │ ON DELETE CASCADE
    ▼
deployment_containers
  id, deployment_id, container_id, host_port (nullable), created_at
```

`deployment_containers` is one row per live container. A deployment whose build only
publishes files deletes its row as soon as the upload finishes; one that leaves an app
running keeps it, and `host_port` is what the site proxy needs. `host_port` is nullable
because it is only known after the app starts listening.

The `install_cmd` / `build_cmd` / `run_cmd` columns hold what the user configured. For a
locked project type they are the stack's own commands (the API overwrites whatever was sent);
for the rest they are the user's, and they are what the build actually runs.

---

## 6. External systems

| System | Reached via | Used for |
|--------|-------------|----------|
| Postgres | `database.DB` (`lib/pq`) | users, deployments, containers |
| Docker daemon | `internal/container`, `docker` CLI over the mounted `/var/run/docker.sock` | build + runtime containers |
| S3 / MinIO | `internal/s3`, AWS SDK v2 (`UsePathStyle` when an endpoint is set) | published build output |
| GitHub | `internal/auth`, `internal/github-repo` (plain `net/http`) | OAuth + repo listing |

The backend container mounts the docker socket and runs `privileged: true`, so build
containers are **siblings** on the host daemon, not children — which is why published ports
land on the docker host and `RUNTIME_HOST=host.docker.internal` is needed inside compose.

User-supplied commands are `eval`ed inside the build container, never on the host. That is
the same trust boundary a `git clone` + `npm install` of an arbitrary repository already
crosses, so making the commands editable does not widen it — but it does mean the container
isolation is the only thing standing between a deployment and the host.

### Environment variables

| Var | Used by | Notes |
|-----|---------|-------|
| `HOST`, `PORT`, `DB_USER`, `DBNAME`, `PASSWORD` | `database` | Postgres DSN, `sslmode=disable` |
| `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRETS` | `pkg/config` | OAuth app |
| `JWT_SECRET` | `pkg/config` | HS256 signing key |
| `FRONTEND_ROUTE` | `pkg/config` | post-login redirect base |
| `AWS_S3_ENDPOINT`, `AWS_S3_BUCKET`, `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` | `internal/s3` | empty endpoint ⇒ real AWS; bucket auto-created |
| `SITE_BASE_DOMAIN`, `SITE_SCHEME` | `internal/site` | default `localhost:3000`, `http` |
| `RUNTIME_HOST` | `internal/site` | where published container ports answer; default `127.0.0.1` |

Note `pkg/config.GithubConfig()` hardcodes `RedirectURL: http://localhost:3000/auth/callback`,
so that one is not yet environment-driven.

---

## 7. Adding a project type

Everything a project type needs lives in one place. Append an entry to `supported` in
`internal/stack/stack.go`:

- `Kind: KindStatic` → set `OutputDirs`; the container exits after the build and the output
  directory is published to object storage.
- `Kind: KindDynamic` → set `RunCmd` and `Port`; the container stays up and gets proxied to,
  and a run command becomes required on the deploy form.
- `Locked: true` → the entry's commands are forced and the form's inputs are read-only.
  `Locked: false` → they are pre-filled defaults the user can edit. React is the only locked
  type today.
- `Enabled: false` → still listed by `GET /stacks`, so the UI can show it as coming soon.

No change is needed in `worker`, `site` or `github-repo` — `Image`, `Kind`, `Locked`, the
commands and `Port` are all read from the table at request and build time.
