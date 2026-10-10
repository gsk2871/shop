import { useEffect, useState, type ReactNode } from "react";
import { history, useLocation, useModel, Outlet } from "@umijs/max";
import { Alert, Spin } from "antd";
import { me } from "../services/auth";
import { APIException } from "../services/cms";
import type { PermissionNode } from "../services/contracts";
function visible(nodes: PermissionNode[], path: string): boolean {
  return nodes.some(
    (node) => node.route === path || visible(node.children, path),
  );
}
export default function Auth({ children }: { children?: ReactNode }) {
  const [ready, setReady] = useState(false);
  const [error, setError] = useState("");
  const location = useLocation();
  const { setInitialState } = useModel("@@initialState");
  useEffect(() => {
    let active = true;
    setReady(false);
    setError("");
    me()
      .then((identity) => {
        if (!active) return;
        void setInitialState({ identity });
        if (
          identity.scope !== "full" &&
          location.pathname !== "/system/security"
        ) {
          history.replace("/system/security");
          return;
        }
        if (
          identity.scope === "full" &&
          !visible(identity.menu, location.pathname)
        ) {
          setError("没有访问此页面的权限");
          return;
        }
        setReady(true);
      })
      .catch((e: Error) => {
        if (!active) return;
        setError(e.message);
        if (e instanceof APIException && e.status === 401)
          history.replace("/login");
      });
    return () => {
      active = false;
    };
  }, [location.pathname]);
  if (error) return <Alert type="error" title={error} />;
  return ready ? <>{children || <Outlet />}</> : <Spin />;
}
