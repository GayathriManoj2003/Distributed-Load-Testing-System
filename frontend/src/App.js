import {
  createBrowserRouter,
  RouterProvider,
  Outlet
} from "react-router-dom"
import React from 'react';
import CreateTest from "./pages/CreateTest";
import Home from "./pages/Home";
// import Navbar from "./components/Navbar";
// import Footer from "./components/Footer";
import './App.css';
import StartTest from "./pages/StartTest";

const Layout = () => {
  return (
    <>
      {/* <Navbar/> */}
      <Outlet/>
      {/* <Footer/> */}
    </>
  )
}

const router = createBrowserRouter([
  {
    path: "/",
    element: <Layout/>,
    children: [
        {
          path: "/submit_test",
          element: <CreateTest/>
        },
        {
          path: "/",
          element: <Home/>
        },
        {
          path: "/start_test",
          element: <StartTest/>
        }
    ]
  }
])
function App() {
  return (
    <div className="App">
      <RouterProvider router = {router}/>
    </div>
  );
}

export default App;