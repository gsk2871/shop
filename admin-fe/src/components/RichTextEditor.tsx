import { useEffect, useRef, useState, type ReactElement } from "react";
import { Button, Input, Modal, Select, Space } from "antd";
import { LexicalComposer } from "@lexical/react/LexicalComposer";
import { RichTextPlugin } from "@lexical/react/LexicalRichTextPlugin";
import { ContentEditable } from "@lexical/react/LexicalContentEditable";
import { HistoryPlugin } from "@lexical/react/LexicalHistoryPlugin";
import { ListPlugin } from "@lexical/react/LexicalListPlugin";
import { LinkPlugin } from "@lexical/react/LexicalLinkPlugin";
import { LexicalErrorBoundary } from "@lexical/react/LexicalErrorBoundary";
import { useLexicalComposerContext } from "@lexical/react/LexicalComposerContext";
import { $generateHtmlFromNodes, $generateNodesFromDOM } from "@lexical/html";
import { $setBlocksType } from "@lexical/selection";
import {
  $createHeadingNode,
  $createQuoteNode,
  HeadingNode,
  QuoteNode,
} from "@lexical/rich-text";
import {
  INSERT_ORDERED_LIST_COMMAND,
  INSERT_UNORDERED_LIST_COMMAND,
  ListNode,
  ListItemNode,
} from "@lexical/list";
import { LinkNode, TOGGLE_LINK_COMMAND } from "@lexical/link";
import {
  DecoratorNode,
  $getRoot,
  $getNodeByKey,
  $getSelection,
  $isRangeSelection,
  $createParagraphNode,
  $insertNodes,
  FORMAT_TEXT_COMMAND,
  UNDO_COMMAND,
  REDO_COMMAND,
  type DOMConversionMap,
  type DOMExportOutput,
  type NodeKey,
  type SerializedLexicalNode,
} from "lexical";
import type { RichText } from "../services/contracts";
import type { Media } from "../services/cms";
import { assetURL } from "../services/cms";
import MediaSelect from "./MediaSelect";
interface SerializedMedia extends SerializedLexicalNode {
  mediaId: string;
  url: string;
  kind: "image";
  alt: string;
}
function MediaView({
  nodeKey,
  id,
  url,
  kind,
  alt,
}: {
  nodeKey: string;
  id: string;
  url: string;
  kind: "image";
  alt: string;
}) {
  const [editor] = useLexicalComposerContext();
  return (
    <span
      contentEditable={false}
      style={{ display: "block", margin: "12px 0" }}
    >
      <img alt={alt} src={assetURL(url)} style={{ maxWidth: "100%", maxHeight: 400 }} />
      <Button
        size="small"
        onClick={() =>
          editor.update(() => {
            $getNodeByKey(nodeKey)?.remove();
          })
        }
      >
        移除素材 #{id}
      </Button>
    </span>
  );
}
class MediaNode extends DecoratorNode<ReactElement> {
  __mediaId: string;
  __url: string;
  __kind: "image";
  __alt: string;
  constructor(
    id: string,
    url: string,
    kind: "image",
    alt = "",
    key?: NodeKey,
  ) {
    super(key);
    this.__mediaId = id;
    this.__url = url;
    this.__kind = kind;
    this.__alt = alt;
  }
  static getType() {
    return "shop-media";
  }
  static clone(node: MediaNode) {
    return new MediaNode(
      node.__mediaId,
      node.__url,
      node.__kind,
      node.__alt,
      node.__key,
    );
  }
  static importJSON(value: Parameters<typeof DecoratorNode.importJSON>[0]) {
    const media = value as unknown as SerializedMedia;
    return new MediaNode(media.mediaId, media.url, media.kind, media.alt);
  }
  exportJSON(): SerializedMedia {
    return {
      ...super.exportJSON(),
      type: "shop-media",
      version: 1,
      mediaId: this.__mediaId,
      url: this.__url,
      kind: this.__kind,
      alt: this.__alt,
    };
  }
  static importDOM(): DOMConversionMap {
    return Object.fromEntries(
      ["img"].map((tag) => [
        tag,
        (dom: HTMLElement) => {
          const id = dom.getAttribute("data-media-id");
          if (!id || !/^\d+$/.test(id)) return null;
          return {
            priority: 2,
            conversion: () => ({
              node: new MediaNode(
                id,
                (dom.getAttribute("src") || "").replace(
                  /^\/backend(?=\/uploads)/,
                  "",
                ),
                "image",
                dom.getAttribute("alt") || "",
              ),
            }),
          };
        },
      ]),
    ) as DOMConversionMap;
  }
  createDOM() {
    return document.createElement("div");
  }
  updateDOM() {
    return false;
  }
  exportDOM(): DOMExportOutput {
    const element = document.createElement(
      "img",
    );
    element.setAttribute("data-media-id", this.__mediaId);
    element.setAttribute("src", this.__url);
    element.setAttribute("alt", this.__alt);
    return { element };
  }
  decorate() {
    return (
      <MediaView
        nodeKey={this.__key}
        id={this.__mediaId}
        url={this.__url}
        kind={this.__kind}
        alt={this.__alt}
      />
    );
  }
}
function Bridge({
  value,
  onChange,
}: {
  value?: RichText;
  onChange?: (value: RichText) => void;
}) {
  const [editor] = useLexicalComposerContext();
  const callback = useRef(onChange);
  callback.current = onChange;
  const last = useRef(value?.html || "");
  const initialized = useRef(false);
  useEffect(() => {
    if (initialized.current && last.current === (value?.html || "")) return;
    initialized.current = true;
    last.current = value?.html || "";
    editor.update(
      () => {
        const root = $getRoot();
        root.clear();
        const parsed = new DOMParser().parseFromString(
          value?.html || "<p></p>",
          "text/html",
        );
        const nodes = $generateNodesFromDOM(editor, parsed);
        if (nodes.length) root.append(...nodes);
        else root.append($createParagraphNode());
      },
      { tag: "external-value" },
    );
  }, [editor, value?.html]);
  useEffect(
    () =>
      editor.registerUpdateListener(({ editorState, tags }) => {
        if (tags.has("external-value")) return;
        editorState.read(() => {
          const html = $generateHtmlFromNodes(editor);
          last.current = html;
          const parsed = new DOMParser().parseFromString(html, "text/html");
          const mediaIds = [
            ...new Set(
              Array.from(parsed.querySelectorAll("[data-media-id]")).map(
                (node) => node.getAttribute("data-media-id") || "",
              ),
            ),
          ];
          callback.current?.({ html, mediaIds });
        });
      }),
    [editor],
  );
  return null;
}
function Toolbar() {
  const [editor] = useLexicalComposerContext();
  const [linkOpen, setLinkOpen] = useState(false);
  const [link, setLink] = useState("");
  function block(value: string) {
    editor.update(() => {
      const selection = $getSelection();
      if ($isRangeSelection(selection))
        $setBlocksType(selection, () =>
          value === "quote"
            ? $createQuoteNode()
            : value === "p"
              ? $createParagraphNode()
              : $createHeadingNode(value as "h2" | "h3" | "h4"),
        );
    });
  }
  function insert(media: Media) {
    editor.update(() => {
      $insertNodes([
        new MediaNode(
          media.id,
          media.url,
          "image",
          media.name,
        ),
        $createParagraphNode(),
      ]);
    });
  }
  return (
    <>
      <Space wrap>
        <Select
          defaultValue="p"
          style={{ width: 100 }}
          options={[
            { value: "p", label: "正文" },
            { value: "h2", label: "二级标题" },
            { value: "h3", label: "三级标题" },
            { value: "h4", label: "四级标题" },
            { value: "quote", label: "引用" },
          ]}
          onChange={block}
        />
        <Button
          onClick={() => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "bold")}
        >
          粗体
        </Button>
        <Button
          onClick={() => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "italic")}
        >
          斜体
        </Button>
        <Button
          onClick={() =>
            editor.dispatchCommand(INSERT_UNORDERED_LIST_COMMAND, undefined)
          }
        >
          无序列表
        </Button>
        <Button
          onClick={() =>
            editor.dispatchCommand(INSERT_ORDERED_LIST_COMMAND, undefined)
          }
        >
          有序列表
        </Button>
        <Button onClick={() => setLinkOpen(true)}>链接</Button>
        <Button
          onClick={() => editor.dispatchCommand(TOGGLE_LINK_COMMAND, null)}
        >
          移除链接
        </Button>
        <MediaSelect  onSelect={insert} />
        <Button onClick={() => editor.dispatchCommand(UNDO_COMMAND, undefined)}>
          撤销
        </Button>
        <Button onClick={() => editor.dispatchCommand(REDO_COMMAND, undefined)}>
          重做
        </Button>
      </Space>
      <Modal
        title="插入链接"
        open={linkOpen}
        onCancel={() => setLinkOpen(false)}
        onOk={() => {
          if (!/^https:\/\/[^\s]+$|^\/(?!\/)[^\s]*$/.test(link)) return;
          editor.dispatchCommand(TOGGLE_LINK_COMMAND, link);
          setLinkOpen(false);
        }}
      >
        <Input
          value={link}
          onChange={(event) => setLink(event.target.value)}
          placeholder="HTTPS 地址或站内路径"
        />
      </Modal>
    </>
  );
}
export default function RichTextEditor({
  value,
  onChange,
}: {
  value?: RichText;
  onChange?: (value: RichText) => void;
}) {
  return (
    <LexicalComposer
      initialConfig={{
        namespace: "shop-editor",
        nodes: [
          HeadingNode,
          QuoteNode,
          ListNode,
          ListItemNode,
          LinkNode,
          MediaNode,
        ],
        onError(error) {
          throw error;
        },
        theme: {
          paragraph: "editor-paragraph",
          text: { bold: "editor-bold", italic: "editor-italic" },
        },
      }}
    >
      <Toolbar />
      <div
        style={{
          position: "relative",
          border: "1px solid #d9d9d9",
          borderRadius: 6,
          marginTop: 12,
        }}
      >
        <RichTextPlugin
          contentEditable={
            <ContentEditable
              aria-label="英文正文"
              style={{ minHeight: 280, padding: 16, outline: "none" }}
            />
          }
          placeholder={
            <div
              style={{
                position: "absolute",
                top: 16,
                left: 16,
                pointerEvents: "none",
                color: "#999",
              }}
            >
              编辑英文内容，图片从媒体库选择
            </div>
          }
          ErrorBoundary={LexicalErrorBoundary}
        />
      </div>
      <Bridge value={value} onChange={onChange} />
      <HistoryPlugin />
      <ListPlugin />
      <LinkPlugin />
    </LexicalComposer>
  );
}
