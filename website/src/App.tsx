import { BrowserRouter, Route, Routes } from "react-router-dom"
import Login from "./pages/Login"
import Landing from "./pages/Landing"

const App = () => {
  return (
    <div className="font-open" >
      <BrowserRouter>
      <Routes>
        <Route path="/" element={<Landing/>} />
        <Route element={<Login/>}  path="/login"/>
      </Routes>
      </BrowserRouter>
    </div>
  )
}

export default App