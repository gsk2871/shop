import { useCallback, useEffect, useRef, useState } from "react";
import { useModel } from "@umijs/max";
import {
  Alert,
  Button,
  Checkbox,
  Empty,
  Image,
  Input,
  Modal,
  Pagination,
  Popconfirm,
  Select,
  Space,
  Spin,
  Table,
  Typography,
  message,
} from "antd";
import {
  FolderOutlined,
  FolderOpenOutlined,
  PlusOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import { api, assetURL, type Media } from "../services/cms";
import { allows } from "../services/auth";
import type { Page } from "../services/contracts";
import { getGroups, type MediaGroup } from "../services/media";
import MediaUpload, { useUploadPolicy } from "./MediaUpload";
import { useMediaUploads } from "./mediaUploadStore";
interface Reference {
  id: string;
  ownerType: string;
  ownerId: string;
  field: string;
}
export default function MediaLibrary({
  onSelect,
}: {
  onSelect?: (media: Media) => void;
}) {
  const { initialState } = useModel("@@initialState");
  const can = (code: string) => allows(initialState?.identity, code);
  const [msg, context] = message.useMessage();
  const { policy, error: policyError } = useUploadPolicy();
  const { revision } = useMediaUploads();
  const [groups, setGroups] = useState<MediaGroup[]>([]);
  const [group, setGroup] = useState("");
  const [q, setQ] = useState("");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [data, setData] = useState<Page<Media>>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 24,
  });
  const [busy, setBusy] = useState(false);
  const [loadError, setLoadError] = useState("");
  const request = useRef(0);
  const [selected, setSelected] = useState<string[]>([]);
  const [edit, setEdit] = useState<{
    kind: "group" | "media";
    id?: string;
    name: string;
  }>();
  const [saving, setSaving] = useState(false);
  const [move, setMove] = useState<string[]>();
  const [target, setTarget] = useState<string>("ungrouped");
  const [refs, setRefs] = useState<Reference[]>();
  const load = useCallback(async () => {
    const seq = ++request.current;
    setBusy(true);
    setLoadError("");
    try {
      const [list, dirs] = await Promise.all([
        api<Page<Media>>(
          `media?page=${page}&pageSize=24&q=${encodeURIComponent(q)}&groupId=${encodeURIComponent(group)}`,
        ),
        getGroups(),
      ]);
      if (seq !== request.current) return;
      setGroups(dirs.items);
      setData(list);
      if (
        group &&
        group !== "ungrouped" &&
        !dirs.items.some((g) => g.id === group)
      ) {
        setGroup("");
        setPage(1);
      } else if (page > 1 && !list.items.length) setPage(page - 1);
    } catch (e) {
      if (seq === request.current) setLoadError((e as Error).message);
    } finally {
      if (seq === request.current) setBusy(false);
    }
  }, [page, q, group]);
  useEffect(() => {
    void load();
    return () => {
      request.current++;
    };
  }, [load, revision]);
  useEffect(() => setSelected([]), [page, q, group, revision]);
  async function perform(action: () => Promise<unknown>, success: string) {
    try {
      await action();
      msg.success(success);
      setSelected([]);
      await load();
    } catch (e) {
      msg.error((e as Error).message);
    }
  }
  const openMove = (ids: string[]) => {
    setMove(ids);
    setTarget(group && group !== "ungrouped" ? group : "ungrouped");
  };
  const currentName =
    group === ""
      ? "全部图片"
      : group === "ungrouped"
        ? "未分组"
        : groups.find((g) => g.id === group)?.name || "目录";
  const directories = [
    { id: "", name: "全部图片" },
    { id: "ungrouped", name: "未分组" },
    ...groups,
  ];
  return (
    <div className="media-library">
      {context}
      <aside className="media-directories">
        <div className="media-directory-title">
          <Typography.Text strong>图片目录</Typography.Text>
          {can("media.update") && (
            <Button
              size="small"
              type="text"
              icon={<PlusOutlined />}
              title="创建目录"
              onClick={() => setEdit({ kind: "group", name: "" })}
            />
          )}
        </div>
        {directories.map((dir) => (
          <div
            className={`media-directory ${group === dir.id ? "active" : ""}`}
            key={dir.id}
          >
            <button
              type="button"
              onClick={() => {
                setGroup(dir.id);
                setPage(1);
              }}
            >
              {group === dir.id ? <FolderOpenOutlined /> : <FolderOutlined />}
              <span>{dir.name}</span>
              {"count" in dir && <small>{String(dir.count)}</small>}
            </button>
            {dir.id && dir.id !== "ungrouped" && can("media.update") && (
              <Space size={0}>
                <Button
                  type="text"
                  size="small"
                  onClick={() =>
                    setEdit({ kind: "group", id: dir.id, name: dir.name })
                  }
                >
                  改名
                </Button>
                <Popconfirm
                  title="删除目录？"
                  description="目录内的图片将移至未分组，不会删除图片。"
                  onConfirm={() =>
                    perform(
                      () => api(`media/groups/${dir.id}`, "DELETE"),
                      "目录已删除",
                    )
                  }
                >
                  <Button type="text" size="small" danger>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            )}
          </div>
        ))}
      </aside>
      <section className="media-content">
        <div className="media-toolbar">
          <div>
            <Typography.Title level={4} style={{ margin: 0 }}>
              {currentName}
            </Typography.Title>
            <Typography.Text type="secondary">
              共 {data.total} 张图片
            </Typography.Text>
          </div>
          <Space wrap>
            <Input.Search
              placeholder="搜索图片名称"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onSearch={(v) => {
                setQ(v.trim());
                setPage(1);
              }}
              allowClear
              style={{ width: 220 }}
            />
            <Button
              icon={<ReloadOutlined />}
              onClick={() => void load()}
              title="刷新"
            />
            {can("media.upload") && (
              <MediaUpload
                groupId={group && group !== "ungrouped" ? group : null}
              />
            )}
          </Space>
        </div>
        {policyError && <Alert type="error" title={policyError} />}
        {policy && !policy.enabled && (
          <Alert
            type="warning"
            title="COS 尚未配置，暂时无法上传新图片。已有图片仍可浏览和选择。"
          />
        )}
        {policy?.enabled && (
          <Typography.Paragraph type="secondary" style={{ marginTop: 12 }}>
            支持 JPEG、PNG、WebP、GIF，单张不超过{" "}
            {(policy.maxImageBytes / 1024 / 1024).toFixed(0)} MiB。
          </Typography.Paragraph>
        )}
        {!onSelect && (
          <Space className="media-selection" wrap>
            <Checkbox
              checked={
                data.items.length > 0 &&
                data.items.every((i) => selected.includes(i.id))
              }
              indeterminate={
                selected.length > 0 &&
                !data.items.every((i) => selected.includes(i.id))
              }
              onChange={(e) =>
                setSelected(e.target.checked ? data.items.map((i) => i.id) : [])
              }
            >
              当页全选
            </Checkbox>
            <span>已选 {selected.length} 张</span>
            {can("media.update") && (
              <Button
                disabled={!selected.length}
                onClick={() => openMove(selected)}
              >
                移动到目录
              </Button>
            )}
            {can("media.delete") && (
              <Popconfirm
                title={`删除选中的 ${selected.length} 张图片？`}
                description="存在引用时整批不删除，请先解除引用。"
                disabled={!selected.length}
                onConfirm={() =>
                  perform(
                    () => api("media/batch-delete", "POST", { ids: selected }),
                    "图片已删除",
                  )
                }
              >
                <Button danger disabled={!selected.length}>
                  批量删除
                </Button>
              </Popconfirm>
            )}
          </Space>
        )}
        {loadError && (
          <Alert
            type="error"
            title={loadError}
            action={<Button onClick={() => void load()}>重试</Button>}
          />
        )}
        <Spin spinning={busy}>
          <Image.PreviewGroup>
            <div className="media-grid">
              {data.items.map((item) => (
                <article
                  className={`media-card ${selected.includes(item.id) ? "selected" : ""}`}
                  key={item.id}
                >
                  <div className="media-thumbnail">
                    <Image
                      src={assetURL(item.url)}
                      alt={item.name}
                      width="100%"
                      height={160}
                      style={{ objectFit: "contain" }}
                    />
                    {!onSelect && (
                      <Checkbox
                        className="media-check"
                        checked={selected.includes(item.id)}
                        onChange={(e) =>
                          setSelected((old) =>
                            e.target.checked
                              ? [...old, item.id]
                              : old.filter((id) => id !== item.id),
                          )
                        }
                      />
                    )}
                  </div>
                  <div className="media-card-body">
                    <Typography.Text
                      strong
                      ellipsis
                      title={item.name}
                      style={{ width: "100%" }}
                    >
                      {item.name}
                    </Typography.Text>
                    <div className="media-meta">
                      {(item.size / 1024 / 1024).toFixed(2)} MiB
                      {item.width && item.height
                        ? ` · ${item.width} × ${item.height}`
                        : ""}
                    </div>
                    <div className="media-meta">
                      {new Date(item.createdAt).toLocaleString("zh-CN", {
                        timeZone: "Asia/Shanghai",
                      })}
                    </div>
                    {onSelect ? (
                      <Button
                        block
                        type="primary"
                        onClick={() => onSelect(item)}
                        style={{ marginTop: 10 }}
                      >
                        选择图片
                      </Button>
                    ) : (
                      <Space wrap size={0}>
                        {can("media.update") && (
                          <>
                            <Button
                              type="link"
                              size="small"
                              onClick={() =>
                                setEdit({
                                  kind: "media",
                                  id: item.id,
                                  name: item.name,
                                })
                              }
                            >
                              重命名
                            </Button>
                            <Button
                              type="link"
                              size="small"
                              onClick={() => openMove([item.id])}
                            >
                              移动
                            </Button>
                          </>
                        )}
                        <Button
                          type="link"
                          size="small"
                          onClick={async () => {
                            try {
                              setRefs(
                                (
                                  await api<{ items: Reference[] }>(
                                    `media/${item.id}/references`,
                                  )
                                ).items,
                              );
                            } catch (e) {
                              msg.error((e as Error).message);
                            }
                          }}
                        >
                          引用位置
                        </Button>
                        {can("media.delete") && (
                          <Popconfirm
                            title="删除图片？"
                            description="被引用图片无法删除。"
                            onConfirm={() =>
                              perform(
                                () => api(`media/${item.id}`, "DELETE"),
                                "图片已删除",
                              )
                            }
                          >
                            <Button type="link" size="small" danger>
                              删除
                            </Button>
                          </Popconfirm>
                        )}
                      </Space>
                    )}
                  </div>
                </article>
              ))}
            </div>
          </Image.PreviewGroup>
          {!busy && !data.items.length && !loadError && (
            <Empty description="此目录暂无图片" style={{ padding: 48 }} />
          )}
        </Spin>
        <Pagination
          current={page}
          pageSize={24}
          total={data.total}
          showSizeChanger={false}
          onChange={setPage}
          style={{ marginTop: 24, textAlign: "right" }}
        />
      </section>
      <Modal
        title={
          edit?.kind === "group"
            ? edit.id
              ? "重命名目录"
              : "创建目录"
            : "重命名图片"
        }
        open={!!edit}
        confirmLoading={saving}
        onCancel={() => setEdit(undefined)}
        onOk={async () => {
          if (!edit?.name.trim()) {
            msg.error("请输入名称");
            return;
          }
          setSaving(true);
          try {
            await api(
              edit.kind === "group"
                ? `media/groups${edit.id ? "/" + edit.id : ""}`
                : `media/${edit.id}`,
              edit.id ? "PATCH" : "POST",
              { name: edit.name.trim() },
            );
            setEdit(undefined);
            await load();
          } catch (e) {
            msg.error((e as Error).message);
          } finally {
            setSaving(false);
          }
        }}
      >
        <Input
          autoFocus
          maxLength={200}
          value={edit?.name}
          onChange={(e) =>
            setEdit((old) => (old ? { ...old, name: e.target.value } : old))
          }
          placeholder="请输入名称"
        />
      </Modal>
      <Modal
        title={`移动 ${move?.length || 0} 张图片`}
        open={!!move}
        confirmLoading={saving}
        onCancel={() => setMove(undefined)}
        onOk={async () => {
          setSaving(true);
          try {
            await api("media/batch-move", "POST", {
              ids: move,
              groupId: target === "ungrouped" ? null : target,
            });
            setMove(undefined);
            setSelected([]);
            await load();
          } catch (e) {
            msg.error((e as Error).message);
          } finally {
            setSaving(false);
          }
        }}
      >
        <Select
          style={{ width: "100%" }}
          value={target}
          onChange={setTarget}
          options={[
            { value: "ungrouped", label: "未分组" },
            ...groups.map((g) => ({ value: g.id, label: g.name })),
          ]}
        />
      </Modal>
      <Modal
        title="图片引用位置"
        open={refs !== undefined}
        onCancel={() => setRefs(undefined)}
        footer={null}
      >
        <Table
          rowKey="id"
          dataSource={refs}
          pagination={false}
          columns={[
            {
              title: "内容类型",
              dataIndex: "ownerType",
              render: (v: string) =>
                ({
                  product: "产品",
                  article: "新闻",
                  banner: "Banner",
                  settings: "品牌设置",
                  whatsapp_account: "客服账号",
                })[v] || v,
            },
            { title: "内容 ID", dataIndex: "ownerId" },
            { title: "字段", dataIndex: "field" },
          ]}
        />
      </Modal>
    </div>
  );
}
