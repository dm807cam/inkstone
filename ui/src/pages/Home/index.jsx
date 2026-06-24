import { Row, Col, Card, Button } from "react-bootstrap";
import { NavLink } from "react-router-dom";
import {
  BsFileEarmarkText,
  BsPhone,
  BsTerminal,
  BsBook,
} from "react-icons/bs";

import branding from "../../common/branding";
import Logo from "../../components/Logo";

const quickLinks = [
  {
    icon: BsFileEarmarkText,
    title: "Your documents",
    body: "Browse, upload and read your notes and PDFs — on desktop or mobile.",
    to: "/documents",
    cta: "Open documents",
    internal: true,
  },
  {
    icon: BsPhone,
    title: "Connect a device",
    body: "Pair your reMarkable tablet with a one-time code to start syncing.",
    to: "/connect",
    cta: "Pair device",
    internal: true,
  },
  {
    icon: BsTerminal,
    title: "Manage from the CLI",
    body: "Use rmapi with RMAPI_HOST pointed at this instance to script your files.",
    href: branding.rmapiUrl,
    cta: "Get rmapi",
  },
  {
    icon: BsBook,
    title: "Documentation",
    body: "Setup, configuration options and the latest project notes.",
    href: branding.docsUrl,
    cta: "Read the docs",
  },
];

const Home = () => {
  return (
    <div className="page">
      <div className="page-narrow">
        <header className="d-flex align-items-center gap-3 mb-3">
          <Logo size={40} />
          <div>
            <h1 className="h3 mb-1">Welcome to {branding.name}</h1>
            <div className="text-secondary">{branding.tagline}</div>
          </div>
        </header>

        <p className="text-secondary mb-4">
          {branding.name} is an unofficial, self-hosted replacement for the
          reMarkable Cloud. Sync and back up your files while keeping full
          control of your hosting environment.
        </p>

        <Row className="g-3">
          {quickLinks.map(({ icon: Icon, title, body, to, href, cta, internal }) => (
            <Col xs={12} sm={6} key={title}>
              <Card className="h-100">
                <Card.Body className="d-flex flex-column">
                  <div className="d-flex align-items-center gap-2 mb-2">
                    <Icon size={20} className="text-secondary" />
                    <Card.Title as="h2" className="h6 mb-0">{title}</Card.Title>
                  </div>
                  <Card.Text className="text-secondary small flex-grow-1">
                    {body}
                  </Card.Text>
                  {internal ? (
                    <Button as={NavLink} to={to} variant="outline-secondary" size="sm" className="align-self-start mt-2">
                      {cta}
                    </Button>
                  ) : (
                    <Button href={href} target="_blank" rel="noreferrer" variant="outline-secondary" size="sm" className="align-self-start mt-2">
                      {cta}
                    </Button>
                  )}
                </Card.Body>
              </Card>
            </Col>
          ))}
        </Row>

        <p className="text-secondary small mt-4 mb-0">
          Tip: documents are uploaded to the folder you have selected. Open the{" "}
          <a href={branding.repoUrl} target="_blank" rel="noreferrer">project on GitHub</a>{" "}
          to follow development.
        </p>
      </div>
    </div>
  );
};

export default Home;
