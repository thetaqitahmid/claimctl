import { useTranslation } from "react-i18next";
import { Menu, MenuButton, MenuItems, MenuItem } from "@headlessui/react";
import { LogOut, ChevronDown, User } from "lucide-react";
import { NavLink, Link, useNavigate } from "react-router-dom";
import { useAppSelector } from "../store/store";
import { useLogoutMutation } from "../store/api/auth";
import { useNotificationContext } from "../hooks/useNotification";

const Navbar = () => {
  const { t } = useTranslation(["components", "common"]);
  const authData = useAppSelector((state) => state.authSlice);
  const navigate = useNavigate();
  const [logout] = useLogoutMutation();
  const { showNotification } = useNotificationContext();

  const logOut = async () => {
    try {
      await logout().unwrap();
      navigate("/login", { replace: true });
    } catch {
      showNotification("error", "Could not sign out. Please try again.");
    }
  };

  return (
    <header className="sticky top-0 z-40 border-b border-slate-800 bg-slate-950">
      <div className="max-w-screen-2xl mx-auto px-4 sm:px-6 py-3 flex flex-wrap items-center justify-between gap-3">
        <Link to="/" className="text-xl font-semibold text-white">
          {t("common:appName")}
        </Link>
        <nav
          aria-label="Main navigation"
          className="flex gap-2 order-3 w-full sm:order-none sm:w-auto"
        >
          <NavLink
            to="/"
            end
            className={({ isActive }) =>
              `rounded-lg px-3 py-2 text-sm ${isActive ? "bg-slate-800 text-white" : "text-slate-300 hover:text-white"}`
            }
          >
            Resources
          </NavLink>
          {authData.role === "admin" && (
            <NavLink
              to="/admin"
              className={({ isActive }) =>
                `rounded-lg px-3 py-2 text-sm ${isActive ? "bg-cyan-500/15 text-cyan-300" : "text-slate-300 hover:text-white"}`
              }
            >
              Administration
            </NavLink>
          )}
        </nav>
        <Menu>
          <MenuButton className="text-slate-300 hover:text-white text-sm flex items-center gap-2 p-2">
            {authData.user || "Account"}
            <ChevronDown className="h-4 w-4" aria-hidden="true" />
          </MenuButton>
          <MenuItems
            anchor="bottom end"
            className="z-50 w-48 rounded-lg border border-slate-600 bg-slate-900 p-1 shadow-xl [--anchor-gap:8px]"
          >
            <MenuItem>
              <Link
                to="/profile"
                className="flex items-center gap-2 px-3 py-2 text-sm text-slate-200 rounded data-[focus]:bg-slate-800"
              >
                <User size={16} />
                {t("components:navbar.profile")}
              </Link>
            </MenuItem>
            <MenuItem>
              <button
                onClick={logOut}
                className="flex w-full items-center gap-2 px-3 py-2 text-sm text-slate-200 rounded data-[focus]:bg-slate-800"
              >
                <LogOut size={16} />
                {t("components:navbar.logout")}
              </button>
            </MenuItem>
          </MenuItems>
        </Menu>
      </div>
    </header>
  );
};

export default Navbar;
