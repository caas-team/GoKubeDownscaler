interface IContextProps {
  children: React.ReactNode;
}

interface IGithubContext {
  /** labels is a static list of available labels on the repository */
  labels: CachedGithubLabel[];
}
