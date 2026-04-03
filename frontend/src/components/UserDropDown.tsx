import { ComponentType, SVGProps, useRef } from "react";
import { useClickOutside } from "../hooks/UseClickOutside";

export interface dropDownProp {
  name: string;
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  action: () => void;
  onclose?: () => void;
  dividerTop?: boolean;
}

export interface UserDropDownProps {
  isOpen: boolean;
  onClose: () => void;
  dropDownPropArray: dropDownProp[];
}

const UserDropDownMenu: React.FC<UserDropDownProps> = ({
  isOpen,
  onClose,
  dropDownPropArray,
}) => {
  const dropDownRef = useRef<HTMLDivElement>(null);

  useClickOutside({
    ref: dropDownRef,
    isOpen,
    onClose,
  });

  return (
    <div
      ref={dropDownRef}
      className={`absolute right-0 top-full mt-2 w-48 rounded-xl shadow-xl bg-slate-900 border border-slate-800/80 z-50 overflow-hidden ${
        isOpen ? "block" : "hidden"
      }`}
    >
      <div className="py-1">
        {dropDownPropArray.map((item) => (
          <div key={item.name}>
            {item.dividerTop && (
              <>
                <div className="h-px bg-slate-800/80 my-1 mx-2"></div>
                <div className="px-4 py-1.5 text-[10px] font-semibold text-slate-500 uppercase tracking-wider">
                  Administration
                </div>
              </>
            )}
            <button
              type="button"
              onClick={() => {
                item.action();
                onClose();
              }}
              className="w-full px-4 py-2.5 text-sm text-slate-300 hover:text-cyan-400 hover:bg-slate-800/50 transition-colors flex items-center gap-3 text-left"
            >
              {item.icon && (
                <item.icon
                  className="w-4 h-4"
                  aria-hidden="true"
                />
              )}
            {item.name}
            </button>
          </div>
        ))}
      </div>
    </div>
  );
};

export default UserDropDownMenu;
