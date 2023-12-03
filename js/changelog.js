import fs from "fs";

const packageJson = fs.readFileSync("./package.json", "utf-8");
const version = JSON.parse(packageJson).version;

fs.readFile("./CHANGELOG.md", "utf-8", (err, data) => {
  if (err) {
    console.error(err);
    return;
  }

  const lines = data.split("\n");

  const changelog = [];
  let match = false;
  for (const line of lines) {
    if (match) {
      if (line.startsWith("## ")) {
        break;
      } else {
        changelog.push(line);
      }
    } else {
      if (line.startsWith(`## ${version}`)) {
        match = true;
        if (line.trim().length > version.length + 3) {
          changelog.push(line);
        }
      }
    }
  }

  fs.writeFile("./CHANGELOG.release.md", changelog.join("\n").trim(), (err) => {
    if (err) {
      console.error(err);
      return;
    }
  });
});
