import { Nav, Navbar, Button, NavDropdown, Container } from "react-bootstrap";
import { NavLink } from "react-router-dom";
import { BsSunFill, BsMoonStarsFill, BsCircleHalf, BsPersonCircle } from "react-icons/bs";

import { logout } from "../common/actions";
import { useAuthState } from "../common/useAuthContext";
import { useTheme } from "../common/useTheme";
import branding from "../common/branding";
import Logo from "./Logo";

const THEME_META = {
  light: { icon: BsSunFill, label: "Light" },
  dark: { icon: BsMoonStarsFill, label: "Dark" },
  system: { icon: BsCircleHalf, label: "System" },
};

function ThemeToggle() {
  const { mode, cycleMode } = useTheme();
  const meta = THEME_META[mode] || THEME_META.system;
  const Icon = meta.icon;
  return (
    <Button
      variant="outline"
      className="d-inline-flex align-items-center"
      onClick={cycleMode}
      title={`Theme: ${meta.label} (click to change)`}
      aria-label={`Theme: ${meta.label}`}
    >
      <Icon />
    </Button>
  );
}

const NavigationBar = () => {
  const { state: { user }, dispatch } = useAuthState();

  function handleLogout() {
    logout(dispatch);
  }

  function isAdmin() {
    return user && user.Roles && user.Roles[0] === "Admin";
  }

  return (
    <Navbar expand="lg" className="app-navbar sticky-top" collapseOnSelect>
      <Container fluid>
        <Navbar.Brand as={NavLink} to="/" className="navbar-brand">
          <Logo size={26} />
          <span>{branding.name}</span>
        </Navbar.Brand>

        {/* Always-visible controls (mobile keeps theme toggle out of the menu) */}
        <div className="d-flex align-items-center gap-2 order-lg-2">
          <ThemeToggle />
          {user && <Navbar.Toggle aria-controls="main-nav" />}
        </div>

        {user && (
          <Navbar.Collapse id="main-nav" className="order-lg-1">
            <Nav className="me-auto">
              <Nav.Link as={NavLink} to="/documents">Documents</Nav.Link>
              <Nav.Link as={NavLink} to="/integrations">Integrations</Nav.Link>
              <Nav.Link as={NavLink} to="/connect">Connect</Nav.Link>
              <Nav.Link as={NavLink} to="/screenshare">Screen Share</Nav.Link>
              {isAdmin() && <Nav.Link as={NavLink} to="/admin">Admin</Nav.Link>}
            </Nav>
            <Nav>
              <NavDropdown
                id="userMenu"
                align="end"
                title={
                  <span className="d-inline-flex align-items-center gap-2">
                    <BsPersonCircle /> {user.UserID}
                  </span>
                }
              >
                <NavDropdown.Item as={NavLink} to="/profile">Profile</NavDropdown.Item>
                <NavDropdown.Divider />
                <NavDropdown.Item as={Button} onClick={handleLogout}>Log out</NavDropdown.Item>
              </NavDropdown>
            </Nav>
          </Navbar.Collapse>
        )}
      </Container>
    </Navbar>
  );
};

export default NavigationBar;
