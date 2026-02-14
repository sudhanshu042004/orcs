import { BrowserRouter, Route, Routes } from "react-router-dom"
import Login from "./pages/Login"
import Landing from "./pages/Landing"
import Signup from "./pages/Signup"

const App = () => {
  return (
    <div className="font-open" >
      <BrowserRouter>
      <Routes>
        <Route path="/" element={<Landing/>} />
        <Route element={<Login/>}  path="/login"/>
        <Route path="/signup" element={<Signup/>} />
      </Routes>
      </BrowserRouter>
    </div>
  )
}

export default App