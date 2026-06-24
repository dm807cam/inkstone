import { useState } from "react";
import { useHistory } from "react-router-dom";
import { Button, Form } from "react-bootstrap";

import { useAuthState } from "../../common/useAuthContext";
import { loginUser } from "../../common/actions";
import branding from "../../common/branding";
import Logo from "../../components/Logo";

import styles from "./Login.module.scss";

const Login = () => {
  let history = useHistory();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");

  const { state, dispatch } = useAuthState();
  const { errorMessage, loading } = state;

  const handleLogin = async (e) => {
    e.preventDefault();

    let payload = { email: username, password };
    try {
      await loginUser(dispatch, payload);
      history.push("/documents");
    } catch (error) {
      console.log(error);
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.card}>
        <div className={styles.brand}>
          <Logo size={44} />
          <div className={styles.brandName}>{branding.name}</div>
          <div className={styles.tagline}>{branding.tagline}</div>
        </div>

        {errorMessage ? <p className={styles.error}>{errorMessage}</p> : null}

        <Form onSubmit={handleLogin}>
          <Form.Group className="mb-3">
            <Form.Label htmlFor="username">Username</Form.Label>
            <Form.Control
              id="username"
              value={username}
              autoFocus
              onChange={(e) => setUsername(e.target.value)}
              disabled={loading}
              placeholder="Username"
              autoComplete="username"
            />
          </Form.Group>

          <Form.Group className="mb-4">
            <Form.Label htmlFor="password">Password</Form.Label>
            <Form.Control
              type="password"
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={loading}
              placeholder="Password"
              autoComplete="current-password"
            />
          </Form.Group>

          <Button type="submit" className="w-100" disabled={loading}>
            {loading ? "Signing in…" : "Sign in"}
          </Button>
        </Form>
      </div>
    </div>
  );
};

export default Login;
