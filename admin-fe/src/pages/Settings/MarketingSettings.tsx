import { Button, Card, Form, Input, Select, Space, Switch } from "antd";
import MediaSelect from "../../components/MediaSelect";

const colorRules = [{ required: true, pattern: /^#[0-9a-fA-F]{6}$/, message: "请选择颜色" }];
const requiredText = (max: number) => [{ required: true, whitespace: true, max }];

export default function MarketingSettings() {
  return <>
    <Card title="页眉与背景图片" style={{ marginBottom: 24 }}>
      <Form.Item name="headerBackground" label="页眉背景色（只影响页眉）" rules={[{ pattern: /^#[0-9a-fA-F]{6}$/ }]}><Input type="color" /></Form.Item>
      <Form.Item name="backgroundMediaId" label="全站背景图片" extra="未配置时使用现有背景。清除后保存即可恢复；不会删除媒体库素材。"><MediaSelect /></Form.Item>
    </Card>
    <Card title="顶部宣传条" style={{ marginBottom: 24 }}>
      <Form.Item name={["announcement", "enabled"]} label="开启宣传条" valuePropName="checked"><Switch /></Form.Item>
      <Space wrap>
        <Form.Item name={["announcement", "background"]} label="背景颜色" rules={colorRules}><Input type="color" /></Form.Item>
        <Form.Item name={["announcement", "textColor"]} label="文字颜色" rules={colorRules}><Input type="color" /></Form.Item>
        <Form.Item name={["announcement", "speed"]} label="滚动速率" rules={[{ required: true }]}><Select style={{ width: 140 }} options={[{ value: "slow", label: "慢" }, { value: "standard", label: "标准" }, { value: "fast", label: "快" }]} /></Form.Item>
      </Space>
      <p>所有内容均为展示文字，不提供点击动作。按列表顺序显示，关闭的条目不展示。</p>
      <Form.List name={["announcement", "items"]}>{(fields, { add, remove, move }) => <>
        {fields.map((field, index) => <Space key={field.key} wrap align="start">
          <Form.Item name={[field.name, "text"]} label={`内容 ${index + 1}`} rules={requiredText(500)}><Input.TextArea maxLength={500} style={{ width: 330 }} /></Form.Item>
          <Form.Item name={[field.name, "enabled"]} label="开启" valuePropName="checked"><Switch /></Form.Item>
          <Button disabled={index === 0} onClick={() => move(index, index - 1)}>上移</Button>
          <Button disabled={index === fields.length - 1} onClick={() => move(index, index + 1)}>下移</Button>
          <Button danger onClick={() => remove(field.name)}>删除</Button>
        </Space>)}
        <div><Button disabled={fields.length >= 30} onClick={() => add({ text: "", enabled: true })}>新增宣传内容</Button></div>
      </>}</Form.List>
    </Card>
    <Card title="首页品牌介绍与指标" style={{ marginBottom: 24 }}>
      <Form.Item name={["brandIntro", "tagline"]} label="顶部标记" rules={[{ max: 200 }]}><Input maxLength={200} /></Form.Item>
      <Form.List name={["brandIntro", "parts"]}>{(fields, { add, remove, move }) => <>
        {fields.map((field, index) => <Space key={field.key} wrap align="start">
          <Form.Item name={[field.name, "text"]} label={`品牌第 ${index + 1} 段`} rules={requiredText(100)}><Input maxLength={100} /></Form.Item>
          <Form.Item name={[field.name, "color"]} label="文字颜色" rules={colorRules}><Input type="color" /></Form.Item>
          <Button disabled={index === 0} onClick={() => move(index, index - 1)}>上移</Button>
          <Button disabled={index === fields.length - 1} onClick={() => move(index, index + 1)}>下移</Button>
          <Button danger onClick={() => remove(field.name)}>删除</Button>
        </Space>)}
        <div><Button disabled={fields.length >= 10} onClick={() => add({ text: "", color: "#2563eb" })}>新增品牌分段</Button></div>
      </>}</Form.List>
      <Form.Item name={["brandIntro", "description"]} label="介绍描述" rules={[{ max: 2000 }]}><Input.TextArea rows={3} maxLength={2000} /></Form.Item>
      <Form.List name={["brandIntro", "metrics"]}>{(fields, { add, remove, move }) => <>
        {fields.map((field, index) => <Space key={field.key} wrap align="start">
          <Form.Item name={[field.name, "value"]} label={`指标 ${index + 1} 数据文本`} rules={requiredText(100)}><Input maxLength={100} placeholder="例如 ≥99%" /></Form.Item>
          <Form.Item name={[field.name, "description"]} label="指标描述" rules={requiredText(200)}><Input maxLength={200} /></Form.Item>
          <Button disabled={index === 0} onClick={() => move(index, index - 1)}>上移</Button>
          <Button disabled={index === fields.length - 1} onClick={() => move(index, index + 1)}>下移</Button>
          <Button danger onClick={() => remove(field.name)}>删除</Button>
        </Space>)}
        <div><Button disabled={fields.length >= 20} onClick={() => add({ value: "", description: "" })}>新增指标</Button></div>
      </>}</Form.List>
      <p>品牌分段和指标按当前顺序展示。两个按钮分别联系客服和打开产品目录。</p>
    </Card>
  </>;
}
