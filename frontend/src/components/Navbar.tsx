import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  LogOut,
  ChevronDown,
  Shield,
  Key,
  Bell,
  Settings,
  User,
} from "lucide-react";
import { useAppSelector } from "../store/store";
import UserDropDownMenu, { dropDownProp } from "./UserDropDown";
import { useNavigate } from "react-router-dom";
import { useLogoutMutation } from "../store/api/auth";

const Navbar = () => {
  const { t } = useTranslation(["components", "common"]);
  const [userDropDown, setUserDropDown] = useState(false);
  const authData = useAppSelector((state) => state.authSlice);
  const navigate = useNavigate();
  const [logout] = useLogoutMutation();

  const logOutAction = useCallback(async () => {
    try {
      await logout().unwrap();
      navigate("/login", { replace: true });
    } catch (error) {
      console.error("Logout failed", error);
    }
  }, [logout, navigate]);

  const adminNavigationAction = useCallback(() => {
    navigate("/admin");
  }, [navigate]);

  const profileNavigationAction = useCallback(() => {
    navigate("/profile");
  }, [navigate]);

  const dropDownPropArray: dropDownProp[] = useMemo(() => {
    const props: dropDownProp[] = [
      {
        name: t("components:navbar.profile"),
        icon: User,
        action: profileNavigationAction,
      },
      {
        name: t("components:navbar.logout"),
        icon: LogOut,
        action: logOutAction,
      },
    ];
    if (authData.role === 'admin') {
      props.push({
        name: t("components:navbar.adminPanel"),
        icon: Shield,
        action: adminNavigationAction,
        dividerTop: true,
      });
      props.push({
        name: t("components:navbar.secrets"),
        icon: Key,
        action: () => navigate("/secrets"),
      });
      props.push({
        name: t("components:navbar.webhooks"),
        icon: Bell,
        action: () => navigate("/webhooks"),
      });
      props.push({
        name: t("components:navbar.settings"),
        icon: Settings,
        action: () => navigate("/admin/settings"),
      });
    }
    return props;
  }, [authData.role, logOutAction, adminNavigationAction, profileNavigationAction, navigate, t]);

  const handleDropDown = () => {
    setUserDropDown((prev) => !prev);
  };

  return (
    <>
      <nav className="glass-panel sticky top-0 z-50 transition-all duration-300">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            {/* Logo */}
            <div className="flex-shrink-0">
              <h1
                onClick={() => navigate("/")}
                className="text-2xl font-light text-white cursor-pointer hover:text-slate-300 transition-colors duration-200"
              >
                {t("common:appName")}
              </h1>
            </div>

            {/* Action Buttons */}
            <div className="flex items-center space-x-6">
              {/* User Menu */}
              <div className="relative">
                <button
                  onClick={handleDropDown}
                  className="text-slate-400 hover:text-white transition-colors duration-200 text-sm flex items-center gap-1"
                >
                  {authData.user || "admin"}
                  <ChevronDown
                    className={`h-3 w-3 transition-transform duration-200 ${userDropDown ? 'rotate-180' : ''}`}
                    aria-hidden="true"
                  />
                </button>
                {userDropDown && (
                  <UserDropDownMenu
                    isOpen={userDropDown}
                    onClose={() => setUserDropDown(false)}
                    dropDownPropArray={dropDownPropArray}
                  />
                )}
              </div>
            </div>
          </div>
        </div>

      </nav>

    </>
  );
};

export default Navbar;
