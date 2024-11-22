(function (Prism) {
  var jsx = Prism.util.clone(Prism.languages.jsx);

  Prism.languages.mdx = Prism.languages.extend("markdown", jsx);
})(Prism);
