import { forwardRef, type ComponentPropsWithoutRef } from "react";
import { Link } from "react-router-dom";

type RouterLinkProps = ComponentPropsWithoutRef<"a">;

const EXTERNAL_HREF = /^([a-z][a-z0-9+.-]*:|\/\/)/i;

/**
 * Bridges Astryx links to react-router. Passed to LinkProvider once, so every
 * Astryx component with an `href` (SideNavItem, Button, Breadcrumbs, ...)
 * navigates client-side. External URLs fall back to a plain <a>.
 */
export const RouterLink = forwardRef<HTMLAnchorElement, RouterLinkProps>(
  function RouterLink({ href = "", ...rest }, ref) {
    if (EXTERNAL_HREF.test(href)) {
      return <a ref={ref} href={href} {...rest} />;
    }
    return <Link ref={ref} to={href} {...rest} />;
  },
);
