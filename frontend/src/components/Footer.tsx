import React from "react";

const Footer: React.FC = () => {
  return (
    <footer className="mt-20 border-t border-slate-800/50 bg-slate-950/50 backdrop-blur-sm py-12">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col md:flex-row justify-between items-center gap-6">
          <div className="flex flex-col items-center md:items-start gap-2">
            <h2 className="text-lg font-bold bg-clip-text text-transparent bg-gradient-to-r from-white to-slate-400">
              claimctl
            </h2>
            <p className="text-slate-500 text-sm">
              &copy; {new Date().getFullYear()} claimctl. All rights reserved.
            </p>
          </div>

          <div className="flex flex-wrap justify-center gap-x-8 gap-y-4">
            <a
              href="#about"
              className="text-slate-400 hover:text-white text-sm transition-colors duration-200"
            >
              About
            </a>
            <a
              href="#contact"
              className="text-slate-400 hover:text-white text-sm transition-colors duration-200"
            >
              Contact
            </a>
            <a
              href="#terms"
              className="text-slate-400 hover:text-white text-sm transition-colors duration-200"
            >
              Terms of Service
            </a>
            <a
              href="#privacy"
              className="text-slate-400 hover:text-white text-sm transition-colors duration-200"
            >
              Privacy Policy
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
};

export default Footer;
