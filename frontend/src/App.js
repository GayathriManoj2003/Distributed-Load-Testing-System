import {
  createBrowserRouter,
  RouterProvider,
  Outlet
} from "react-router-dom"

import React from 'react';
import Home from "./pages/Home";
import SubmitTest from "./pages/SubmitTest";
import Dashboard from "./pages/Dashboard";
import './App.css';

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
          element: <SubmitTest/>
        },
        {
          path: "/",
          element: <Home/>
        },
        {
          path: "/dashboard",
          element: <Dashboard/>
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