import { useState } from "react";
import { NavLink, Outlet } from "react-router-dom";
import { Menu, X } from "lucide-react";

const sections = [
  {
    title: "People and access",
    links: [
      ["Users", "/admin/users"],
      ["Groups", "/admin/groups"],
      ["Spaces", "/admin/spaces"],
    ],
  },
  {
    title: "Integrations",
    links: [
      ["Webhooks", "/admin/webhooks"],
      ["Secrets", "/admin/secrets"],
    ],
  },
  {
    title: "System Settings",
    links: [
      ["Authentication", "/admin/settings/authentication"],
      ["Notifications", "/admin/settings/notifications"],
    ],
  },
];

const AdminLayout = () => {
  const [open, setOpen] = useState(false);

  return (
    <div className="max-w-screen-2xl mx-auto lg:flex">
      <div className="lg:hidden p-4 border-b border-slate-800">
        <button
          className="btn-ghost w-full"
          aria-expanded={open}
          aria-controls="admin-navigation"
          onClick={() => setOpen(!open)}
        >
          {open ? <X size={18} /> : <Menu size={18} />} Administration
          navigation
        </button>
      </div>
      <aside
        id="admin-navigation"
        className={`${open ? "block" : "hidden"} lg:block lg:w-64 shrink-0 border-r border-slate-800 bg-slate-900 p-5`}
      >
        <nav
          aria-label="Administration"
          className="lg:sticky lg:top-24 space-y-6"
        >
          <h2 className="text-lg font-semibold text-white">Administration</h2>
          {sections.map((section) => (
            <div key={section.title}>
              <p className="text-xs font-semibold text-slate-400 mb-2 uppercase tracking-wide">
                {section.title}
              </p>
              <div className="space-y-1">
                {section.links.map(([label, path]) => (
                  <NavLink
                    key={path}
                    to={path}
                    onClick={() => setOpen(false)}
                    className={({ isActive }) =>
                      `block rounded-lg px-3 py-2.5 text-sm ${isActive ? "bg-cyan-500/15 text-cyan-300 font-semibold" : "text-slate-300 hover:bg-slate-800 hover:text-white"}`
                    }
                  >
                    {label}
                  </NavLink>
                ))}
              </div>
            </div>
          ))}
        </nav>
      </aside>
      <div className="flex-1 min-w-0 p-4 sm:p-6 lg:p-8">
        <Outlet />
      </div>
    </div>
  );
};

export default AdminLayout;
