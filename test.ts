import { marked } from "marked";
import { transformGfm } from "./index.js";

import { readFileSync } from "fs";

let start = performance.now();
const markdown = readFileSync("node_modules/marked/README.md", "utf-8");

// for (let i = 0; i < 1000; i++) {
//   marked(markdown);
// }
// start = performance.now() - start;
// console.log(start);
// for (let i = 0; i < 1000; i++) {
//   transformGfm(markdown)
// }
// start = performance.now() - start;
// console.log(start);
const htm = transformGfm(markdown);
console.log(htm)