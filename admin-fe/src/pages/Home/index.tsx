import { useEffect } from "react";
import { history, useModel } from "@umijs/max";
import { Spin } from "antd";
import type { PermissionNode } from "../../services/contracts";

import { groupPermissionMenus } from "../../navigation";

function firstRoute(nodes: PermissionNode[]): string | undefined {
  for (const node of nodes) {
    if (node.type === "menu" && node.route && node.route !== "/overview")
      return node.route;
    const child = firstRoute(node.children);
    if (child) return child;
  }
}
export default function Home() {
  const { initialState } = useModel("@@initialState");
  useEffect(() => {
    const identity = initialState?.identity;
    history.replace(
      !identity
        ? "/login"
        : identity.scope !== "full"
          ? "/system/security"
          : firstRoute(groupPermissionMenus(identity.menu)) || "/system/security",
    );
  }, [initialState]);
  return <Spin />;
}
