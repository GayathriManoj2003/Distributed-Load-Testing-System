import {
  createBrowserRouter,
  RouterProvider
} from "react-router-dom"

import Home from "./pages/Home";
import SubmitTest from "./pages/SubmitTest";
import Dashboard from "./pages/Dashboard";
import './App.css';
import { TestIDProvider } from "./context/TestIDContext";

const router = createBrowserRouter([
	{
	  path: "/",
	  children: [
		  {
			path: "/submit_test",
			element:
					<TestIDProvider>
						<SubmitTest/>
					</TestIDProvider>
		  },
		  {
			path: "/",
			element: <Home/>
		  },
		  {
			path: "/dashboard",
			element:
					<TestIDProvider>
						<Dashboard/>
					</TestIDProvider>
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