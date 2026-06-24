import { useState, useLayoutEffect } from "react";
import apiservice from "../../services/api.service"
import Button from 'react-bootstrap/Button';
import { FaRepeat } from "react-icons/fa6";

import styles from "./Connect.module.scss";

export default function CodeGenerator() {

  const [code, setCode] = useState("")
  const [error, setError] = useState("")

  const newCode = async () => {
    setCode("")
    const code = await apiservice.getCode()
      .catch(e => {
        setError(e)
      })
    setCode(code)
  }

  useLayoutEffect(() => {
    newCode()
  }, [])

  if (error) {
    return (
      <div className="page">
        <div className="page-narrow text-center text-danger">{error.message}</div>
      </div>
    );
  }

  return (
    <div className="page">
      <div className={styles.wrap}>
        <h3 className="mb-1">Connect a device</h3>
        <p className="text-secondary mb-4">
          On your reMarkable, choose to connect to the cloud and enter this
          one-time code.
        </p>

        <div className={styles.codeCard}>
          <div className={styles.code}>
            {code ? code : <span className={styles.codePlaceholder}>······</span>}
          </div>
        </div>

        <Button onClick={newCode} variant="outline-secondary" className="d-inline-flex align-items-center gap-2">
          <FaRepeat /> New code
        </Button>
      </div>
    </div>
  );
}
