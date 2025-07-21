import test from "ava";
import { transform, transformGfm, transformMdx } from "../index.js";

test("Markdown Conversion: Paragraphs", async (t) => {
  const input = "This is a regular paragraph";
  const expected = "<p>This is a regular paragraph</p>";
  t.is(transform(input), expected);
});

test("Markdown Conversion: Headers", async (t) => {
  const input = "# Header 1\n## Header 2";
  const expected = /<h1>Header 1<\/h1>\s*<h2>Header 2<\/h2>/s;
  t.regex(transform(input), expected);
});

test("Markdown Conversion: Emphasis", async (t) => {
  const input = "*italic* and **bold** text";
  const expected = /<em>italic<\/em> and <strong>bold<\/strong> text/;
  t.regex(transform(input), expected);
});

test("Markdown Conversion: Unordered lists", async (t) => {
  const input = "- Item 1\n- Item 2";
  const expected = /<ul>\s*<li>Item 1<\/li>\s*<li>Item 2<\/li>\s*<\/ul>/s;
  t.regex(transform(input), expected);
});

test("Markdown Conversion: Ordered lists", async (t) => {
  const input = "1. First\n2. Second";
  const expected = /<ol>\s*<li>First<\/li>\s*<li>Second<\/li>\s*<\/ol>/s;
  t.regex(transform(input), expected);
});

test("Markdown Conversion: Links", async (t) => {
  const input = "[Example](https://example.com)";
  const expected = /<a href=\"https:\/\/example\.com">Example<\/a><\/p>/;
  t.regex(transform(input), expected);
});

test("Markdown Conversion: Images", async (t) => {
  const input = "![Alt text](/path/to/img.jpg)";
  const expected = /<img src=\"\/path\/to\/img.jpg\"/;
  t.regex(transform(input), expected);
});

test("Markdown Conversion: Inline code", async (t) => {
  const input = "Use `console.log()` for debugging";
  const expected = "<p>Use <code>console.log()</code> for debugging</p>";
  t.is(transform(input), expected);
});

test("Markdown Conversion: Code blocks", async (t) => {
  const input = "```\nconst x = 1;\n```";
  const expected = /<pre><code>const x = 1;\n<\/code><\/pre>/s;
  t.regex(transform(input), expected);
});

test("Markdown Conversion: Blockquotes", async (t) => {
  const input = "> This is a blockquote";
  const expected = /<blockquote>\s*<p>This is a blockquote<\/p>\s*<\/blockquote>/s;
  t.regex(transform(input), expected);
});

test("Markdown Conversion: Horizontal rule", async (t) => {
  const input = "---";
  const expected = /<hr/;
  t.regex(transform(input), expected);
});

test("GFM Conversion: Basic paragraph", async (t) => {
  const input = "This is a **bold** paragraph";
  const expected = /This is a <strong>bold<\/strong> paragraph/;
  t.regex(transformGfm(input), expected);
});

test("GFM Conversion: Tables", async (t) => {
  const input = `
| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |
`;
  const expected = /<table>.*<th>Header 1<\/th>.*<td>Cell 1<\/td>/s;
  t.regex(transformGfm(input), expected);
});

test("GFM Conversion: Task lists", async (t) => {
  const input = `
- [x] Task 1
- [ ] Task 2
`;
  const expected = /<input type="checkbox" disabled=""/;
  t.regex(transformGfm(input), expected);
});

test("GFM Conversion: Fenced code blocks", async (t) => {
  const input = "```js\nconsole.log('hello')\n```";
  const expected = /<pre><code class="language-js">.*console\.log/s;
  t.regex(transformGfm(input), expected);
});

test("GFM Conversion: Autolinks", async (t) => {
  const input = "https://github.com";
  t.regex(transformGfm(input), /<a href="https:\/\/github\.com">https:\/\/github\.com<\/a>/);
});

test("GFM Conversion: Strikethrough", async (t) => {
  const input = "~~strikethrough~~";
  t.regex(transformGfm(input), /<del>strikethrough<\/del>/);
});

test("MDX Conversion: Standard Markdown paragraphs", async (t) => {
  const input = "This is a **bold** paragraph";
  const mdx = transformMdx(input);
  t.regex(mdx, /function MDXContent/);
  t.regex(mdx, /export default MDXContent/);
});

test("MDX Conversion: JSX components", async (t) => {
  const input = `import Button from './Button';\n\n<Button onClick={_onClick}>Click me</Button>`;
  const mdx = transformMdx(input);
  console.log(mdx)
  t.regex(mdx, /_jsx\(Button/);
  t.regex(mdx, /onClick: _onClick/);
  t.regex(mdx, /children: \"Click me\"/);
});
