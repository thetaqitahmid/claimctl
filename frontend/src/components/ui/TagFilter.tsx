import { useMemo, useState, useRef, useEffect } from "react";
import { X, ChevronDown, Tags } from "lucide-react";

interface TagFilterProps {
  allTags: string[];
  selectedTags: string[];
  onChange: (tags: string[]) => void;
}

const TagFilter = ({ allTags, selectedTags, onChange }: TagFilterProps) => {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [isOpen]);

  const handleToggleTag = (tag: string) => {
    if (selectedTags.includes(tag)) {
      onChange(selectedTags.filter((t) => t !== tag));
    } else {
      onChange([...selectedTags, tag]);
    }
  };

  const handleRemoveTag = (tag: string, e: React.MouseEvent) => {
    e.stopPropagation();
    onChange(selectedTags.filter((t) => t !== tag));
  };

  const handleClearAll = (e: React.MouseEvent) => {
    e.stopPropagation();
    onChange([]);
  };

  const sortedTags = useMemo(() => {
    return [...allTags].sort((a, b) => a.localeCompare(b));
  }, [allTags]);

  return (
    <div className="relative w-full" ref={dropdownRef}>
      {/* Selected Tags Display */}
      {selectedTags.length > 0 && (
        <div className="flex flex-wrap gap-1 mb-2">
          {selectedTags.map((tag) => (
            <span
              key={tag}
              className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium bg-cyan-600/20 text-cyan-400 border border-cyan-600/30"
            >
              {tag}
              <button
                onClick={(e) => handleRemoveTag(tag, e)}
                className="hover:text-cyan-300 transition-colors"
                title={`Remove ${tag}`}
              >
                <X className="w-3 h-3" />
              </button>
            </span>
          ))}
          <button
            onClick={handleClearAll}
            className="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-medium text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
            title="Clear all tags"
          >
            Clear all
          </button>
        </div>
      )}

      {/* Dropdown Trigger */}
      <button
        onClick={(e) => {
          e.stopPropagation();
          setIsOpen(!isOpen);
        }}
        className="w-full flex items-center justify-between gap-2 rounded bg-slate-800/50 px-2 py-1 text-xs text-slate-200 border border-slate-700 hover:border-cyan-500 focus:outline-none focus:border-cyan-500 font-normal transition-colors"
      >
        <span className="flex items-center gap-1 truncate">
          <Tags className="w-3 h-3" />
          {selectedTags.length > 0
            ? `${selectedTags.length} tag${selectedTags.length > 1 ? "s" : ""} selected`
            : "Filter by tags..."}
        </span>
        <ChevronDown className={`w-3 h-3 transition-transform ${isOpen ? "rotate-180" : ""}`} />
      </button>

      {/* Dropdown Menu */}
      {isOpen && (
        <div className="absolute z-50 mt-1 w-full max-h-64 overflow-auto rounded-lg bg-slate-900 border border-slate-700 shadow-2xl">
          {sortedTags.length === 0 ? (
            <div className="px-3 py-2 text-xs text-slate-500 italic">
              No tags available
            </div>
          ) : (
            <div className="py-1">
              {sortedTags.map((tag) => {
                const isSelected = selectedTags.includes(tag);
                return (
                  <button
                    key={tag}
                    onClick={(e) => {
                      e.stopPropagation();
                      handleToggleTag(tag);
                    }}
                    className="w-full flex items-center gap-2 px-3 py-2 text-xs hover:bg-slate-800/50 transition-colors text-left"
                  >
                    <div
                      className={`w-4 h-4 rounded border-2 flex items-center justify-center flex-shrink-0 transition-colors ${
                        isSelected
                          ? "bg-cyan-600 border-cyan-600"
                          : "border-slate-600"
                      }`}
                    >
                      {isSelected && (
                        <svg
                          className="w-3 h-3 text-white"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={3}
                            d="M5 13l4 4L19 7"
                          />
                        </svg>
                      )}
                    </div>
                    <span className={isSelected ? "text-cyan-400 font-medium" : "text-slate-300"}>
                      {tag}
                    </span>
                  </button>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default TagFilter;
