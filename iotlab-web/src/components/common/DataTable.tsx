import { useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Table, type TableColumn, type TablePlugin } from "@astryxdesign/core/Table";
import { Pagination } from "@astryxdesign/core/Pagination";
import { EmptyState } from "@astryxdesign/core/EmptyState";
import { VStack } from "@astryxdesign/core/Stack";
import { Section } from "@astryxdesign/core/Section";

interface DataTableProps<T extends Record<string, unknown>> {
  /** Rows AFTER filtering/sorting — this component only paginates. */
  data: T[];
  columns: TableColumn<T>[];
  idKey: (keyof T & string) | ((item: T) => string | number);
  /** Rendered above the table — typically <TableToolbar>. */
  toolbar?: ReactNode;
  /** Astryx table plugins (sorting, selection, ...), passed straight through. */
  plugins?: Record<string, TablePlugin<T>>;
  pageSize?: number;
  emptyTitle?: string;
  emptyDescription?: string;
  /** e.g. a "Clear filters" button. */
  emptyActions?: ReactNode;
  density?: "compact" | "balanced" | "spacious";
}

/**
 * Data-list building block: toolbar slot + Table + client-side pagination +
 * empty state. Filtering/sorting stay in the page (they are data concerns);
 * presentation concerns live here. For server-side pagination, lift `page`
 * up and slice on the server instead.
 */
export function DataTable<T extends Record<string, unknown>>({
  data,
  columns,
  idKey,
  toolbar,
  plugins,
  pageSize = 10,
  emptyTitle,
  emptyDescription,
  emptyActions,
  density = "balanced",
}: DataTableProps<T>) {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);

  // Clamp instead of resetting in an effect: when filters shrink the data,
  // the current page may fall past the end.
  const totalPages = Math.max(1, Math.ceil(data.length / pageSize));
  const safePage = Math.min(page, totalPages);
  const pageData = data.slice((safePage - 1) * pageSize, safePage * pageSize);

  return (
    <VStack gap={0}>
      {toolbar}
      {data.length === 0 ? (
        <Section variant="transparent" paddingBlock={10}>
          <EmptyState
            title={emptyTitle ?? t("common.noResults")}
            description={emptyDescription ?? t("common.noMatchDescription")}
            actions={emptyActions}
          />
        </Section>
      ) : (
        <>
          <Table
            data={pageData}
            columns={columns}
            idKey={idKey}
            plugins={plugins}
            density={density}
            hasHover
          />
          {data.length > pageSize ? (
            <Section variant="transparent" paddingBlock={2}>
              <Pagination
                page={safePage}
                onChange={setPage}
                totalItems={data.length}
                pageSize={pageSize}
                variant="count"
                size="sm"
              />
            </Section>
          ) : null}
        </>
      )}
    </VStack>
  );
}
