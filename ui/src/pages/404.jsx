import { Button } from "react-bootstrap";
import { NavLink } from "react-router-dom";

const NoMatch = () => {
  return (
    <div className="page">
      <div className="page-narrow text-center" style={{ paddingTop: "12vh" }}>
        <div className="display-4 fw-semibold mb-2">404</div>
        <p className="text-secondary mb-4">We couldn’t find that page.</p>
        <Button as={NavLink} to="/" variant="outline-secondary">Go home</Button>
      </div>
    </div>
  );
};

export default NoMatch;
