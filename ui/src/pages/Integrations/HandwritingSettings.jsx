import { useState, useEffect } from "react";
import Form from "react-bootstrap/Form";
import Button from "react-bootstrap/Button";
import Card from "react-bootstrap/Card";
import { toast } from "react-toastify";

import apiService from "../../services/api.service";

const emptyForm = {
  provider: "",
  llmUrl: "",
  llmKey: "",
  llmModel: "",
  llmPrompt: "",
  langOverride: "",
};

export default function HandwritingSettings() {
  const [form, setForm] = useState(emptyForm);
  const [hasKey, setHasKey] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    apiService
      .getHwr()
      .then((s) => {
        setForm({
          provider: s.provider || "",
          llmUrl: s.llmUrl || "",
          llmKey: "",
          llmModel: s.llmModel || "",
          llmPrompt: s.llmPrompt || "",
          langOverride: s.langOverride || "",
        });
        setHasKey(!!s.hasKey);
      })
      .catch((e) => setError(e.toString()));
  }, []);

  const enabled = form.provider === "llm";

  function handleChange({ target }) {
    setForm({ ...form, [target.name]: target.value });
  }

  function toggleEnabled({ target }) {
    setForm({ ...form, provider: target.checked ? "llm" : "" });
  }

  async function handleSubmit(event) {
    event.preventDefault();
    setError("");
    try {
      await apiService.saveHwr(form);
      if (form.llmKey) setHasKey(true);
      setForm((f) => ({ ...f, llmKey: "" }));
      toast("Saved");
    } catch (e) {
      setError(e.toString());
    }
  }

  return (
    <>
      <h3 className="mb-3 mt-4">Handwriting recognition</h3>
      <Card>
        <Card.Body>
          <p className="text-secondary small">
            Use a vision LLM (Ollama, OpenRouter, OpenAI, ...) to convert your handwriting to text
            directly from the tablet. Leave disabled to use the server default.
          </p>
          <Form onSubmit={handleSubmit} autoComplete="off">
            <Form.Group className="mb-3" controlId="hwrEnabled">
              <Form.Check
                type="switch"
                label="Use my own vision LLM"
                checked={enabled}
                onChange={toggleEnabled}
              />
            </Form.Group>
            {enabled && (
              <>
                <Form.Group className="mb-3" controlId="hwrUrl">
                  <Form.Label>Base URL</Form.Label>
                  <Form.Control
                    name="llmUrl"
                    placeholder="http://localhost:11434/v1"
                    value={form.llmUrl}
                    onChange={handleChange}
                  />
                  <Form.Text className="text-secondary">
                    OpenAI-compatible endpoint, e.g. Ollama, OpenRouter, or OpenAI.
                  </Form.Text>
                </Form.Group>
                <Form.Group className="mb-3" controlId="hwrModel">
                  <Form.Label>Model</Form.Label>
                  <Form.Control
                    name="llmModel"
                    placeholder="llama3.2-vision"
                    value={form.llmModel}
                    onChange={handleChange}
                  />
                </Form.Group>
                <Form.Group className="mb-3" controlId="hwrKey">
                  <Form.Label>API key</Form.Label>
                  <Form.Control
                    name="llmKey"
                    type="password"
                    placeholder={
                      hasKey ? "•••••••• (leave blank to keep)" : "optional for local Ollama"
                    }
                    value={form.llmKey}
                    onChange={handleChange}
                    autoComplete="new-password"
                  />
                </Form.Group>
                <Form.Group className="mb-3" controlId="hwrPrompt">
                  <Form.Label>Prompt (optional)</Form.Label>
                  <Form.Control
                    as="textarea"
                    rows={2}
                    name="llmPrompt"
                    placeholder="Override the default transcription prompt"
                    value={form.llmPrompt}
                    onChange={handleChange}
                  />
                </Form.Group>
                <Form.Group className="mb-3" controlId="hwrLang">
                  <Form.Label>Language hint (optional)</Form.Label>
                  <Form.Control
                    name="langOverride"
                    placeholder="e.g. en, de, zh_CN"
                    value={form.langOverride}
                    onChange={handleChange}
                  />
                </Form.Group>
              </>
            )}
            {error && <div className="alert alert-danger">{error}</div>}
            <Button variant="primary" type="submit">
              Save
            </Button>
          </Form>
        </Card.Body>
      </Card>
    </>
  );
}
