import { useEffect } from "react";
import { BrowserRouter as Router, Route, Switch, Redirect } from "react-router-dom";
import { ToastContainer } from 'react-toastify';

import apiService from "./services/api.service";
import { AuthProvider } from "./common/useAuthContext";
import { ThemeProvider, useTheme } from "./common/useTheme";
import Role from "./common/Role";
import { PrivateRoute } from "./components/PrivateRoute";
import Navigationbar from "./components/Navigation";
import PasscodeResets from "./components/PasscodeResets";

import Login from "./pages/Login";
import Connect from "./pages/Connect";
import Documents from "./pages/Documents";
import Integrations from "./pages/Integrations";
import Profile from "./pages/Profile";
import Admin from "./pages/Admin";
import ScreenShare from "./pages/ScreenShare";
import NoMatch from "./pages/404";

import "react-toastify/dist/ReactToastify.css";

import "./App.scss"

import { pdfjs } from "react-pdf";
pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  'pdfjs-dist/build/pdf.worker.min.mjs',
  import.meta.url,
).toString();

function ThemedToasts() {
  const { resolved } = useTheme();
  return <ToastContainer autoClose={2000} theme={resolved} position="bottom-right" />;
}

export default function App() {

  useEffect(() => {
    apiService.checkLogin()
  }, [])

  return (
    <ThemeProvider>
      <AuthProvider>
        <Router>
          <div className="app-shell">
            <Navigationbar />
            <PasscodeResets />
            <main className="app-main">
              <Switch>
                {/* Land straight on the documents view instead of a separate start screen. */}
                <Route exact path="/"><Redirect to="/documents" /></Route>
                <PrivateRoute path="/documents/:itemId?" component={Documents} />
                <PrivateRoute path="/connect" component={Connect} />
                <PrivateRoute path="/pair/app" component={Connect} />
                <PrivateRoute path="/pair" component={Connect} />
                <PrivateRoute path="/integrations" component={Integrations} />
                <PrivateRoute path="/profile" component={Profile} />
                <PrivateRoute path="/screenshare" component={ScreenShare} />
                <PrivateRoute path="/admin" roles={[Role.Admin]} component={Admin} />

                <Route path="/login" component={Login} />
                <Route component={NoMatch} />
              </Switch>
            </main>
          </div>
        </Router>
        <ThemedToasts />
      </AuthProvider>
    </ThemeProvider>
  );
}
