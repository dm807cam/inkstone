import Stack from "react-bootstrap/Stack";
import { useAuthState } from "../../common/useAuthContext";

import ResetPassword from "./ResetPassword";
import HandwritingSettings from "./HandwritingSettings";

const Profile = () => {
  const { state: { user } } = useAuthState();
  return (
    <div className="page">
      <div className="page-narrow">
        <h3 className="mb-3">Profile</h3>
        <Stack gap={3}>
          {user.scopes === "sync15" && (
            <div className="text-secondary small">Using sync 15</div>
          )}
          <ResetPassword />
          <hr />
          <HandwritingSettings />
        </Stack>
      </div>
    </div>
  );
};

export default Profile;
