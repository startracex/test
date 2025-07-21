#![deny(clippy::all)]

// use mdxjs::JsxRuntime;
// use napi::{Error, Result, Status};

#[macro_use]
extern crate napi_derive;

#[napi]
pub fn echo()  {
  
}

// #[napi]
// pub fn transform(code: String) -> Result<String> {
//   match markdown::to_html_with_options(&code, &markdown::Options::default()) {
//     Ok(result) => Ok(result),
//     Err(e) => Err(Error::new(Status::InvalidArg, e.reason)),
//   }
// }

// #[napi]
// pub fn transform_gfm(code: String) -> Result<String> {
//   match markdown::to_html_with_options(&code, &markdown::Options::gfm()) {
//     Ok(result) => Ok(result),
//     Err(e) => Err(Error::new(Status::InvalidArg, e.reason)),
//   }
// }

// #[napi(object)]
// pub struct MDXOptions {
//   pub development: Option<bool>,
//   pub provider_import_source: Option<String>,
//   pub jsx: Option<bool>,
//   pub jsx_import_source: Option<String>,
//   pub pragma: Option<String>,
//   pub pragma_frag: Option<String>,
//   pub pragma_import_source: Option<String>,
//   pub filepath: Option<String>,
//   #[napi(ts_type = "\"classic\" | \"automatic\"")]
//   pub jsx_runtime: Option<String>,
// }

// #[napi]
// pub fn transform_mdx(code: String, options: Option<MDXOptions>) -> Result<String> {
//   let compile_options = options.map_or_else(
//     || mdxjs::Options::default(),
//     |options| mdxjs::Options {
//       development: options.development.unwrap_or_default(),
//       jsx_import_source: options.jsx_import_source,
//       jsx: options.jsx.unwrap_or_default(),
//       jsx_runtime: if options.jsx_runtime.as_deref() == Some("classic") {
//         Some(JsxRuntime::Classic)
//       } else {
//         Some(JsxRuntime::Automatic)
//       },
//       parse: mdxjs::MdxParseOptions::default(),
//       provider_import_source: options.provider_import_source,
//       pragma: options.pragma,
//       pragma_frag: options.pragma_frag,
//       pragma_import_source: options.pragma_import_source,
//       filepath: options.filepath,
//     },
//   );

//   match mdxjs::compile(&code, &compile_options) {
//     Ok(result) => Ok(result),
//     Err(e) => Err(Error::new(Status::InvalidArg, e.reason)),
//   }
// }
