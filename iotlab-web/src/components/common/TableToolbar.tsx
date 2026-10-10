import type { ReactNode } from "react";
import { Toolbar } from "@astryxdesign/core/Toolbar";
import { TextInput } from "@astryxdesign/core/TextInput";
import { MagnifyingGlassIcon } from "@heroicons/react/24/outline";

interface TableToolbarProps {
  label: string;
  search: string;
  onSearchChange: (value: string) => void;
  searchPlaceholder?: string;
  /** Right-aligned content (bulk actions, MoreMenu, ...). */
  actions?: ReactNode;
}

/** Standard toolbar above a DataTable: search + actions. */
export function TableToolbar({
  label,
  search,
  onSearchChange,
  searchPlaceholder = "Search...",
  actions,
}: TableToolbarProps) {
  return (
    <Toolbar
      label={label}
      size="sm"
      dividers={["bottom"]}
      startContent={
        <TextInput
          label="Search"
          isLabelHidden
          size="sm"
          hasClear
          placeholder={searchPlaceholder}
          value={search}
          onChange={onSearchChange}
          startIcon={MagnifyingGlassIcon}
        />
      }
      endContent={actions}
    />
  );
}
