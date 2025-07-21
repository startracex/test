use mdxjs::{JsxRuntime, Options};

fn main() -> Result<(), markdown::message::Message> {
  let mut o: Options = Default::default();
  // o.jsx = true;
  o.pragma_frag = Some("xxa".to_string());
  o.pragma_import_source=Some("xxa".to_string());
  o.jsx_runtime = Some(JsxRuntime::Classic);
  println!(
    "{}",
    mdxjs::compile(
      r###"
import {Chart} from './snowfall.js'
export const year = 2018

# Last year’s snowfall

<>hi</>

In {year}, the snowfall was above average.
It was followed by a warm spring which caused
flood conditions in many of the nearby rivers.

<Chart year={year} color="#fcb32c" />
"###,
      &o
    )?
  );

  Ok(())
}
