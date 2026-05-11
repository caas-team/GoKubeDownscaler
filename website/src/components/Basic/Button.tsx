import Link from "@docusaurus/Link";

type ButtonProps = {
  name: string;
  to: string;
  className?: string;
  primary?: boolean;
};

type GitHubButtonProps = {
  href?: string;
  label?: string;
  className?: string;
  primary?: boolean;
};

export function GitHubButton({
  href = "https://github.com/caas-team/GoKubeDownscaler",
  label = "Star on GitHub",
  className,
  primary = false,
}: GitHubButtonProps) {
  const baseClasses =
    "rounded-md cursor-pointer text-xl font-bold py-2 px-8 text-center duration-200 transition-colors select-none whitespace-nowrap no-underline hover:no-underline inline-flex items-center justify-center gap-2 border border-solid w-full";

  const variantClasses = primary
    ? "bg-magenta hover:bg-magenta-hover active:bg-magenta-active border-magenta hover:border-magenta-hover active:border-magenta-active text-white hover:text-white dark:bg-gray-100 dark:hover:bg-gray-200 dark:active:bg-gray-300 dark:border-gray-100 dark:hover:border-gray-200 dark:text-slate-900 dark:hover:text-slate-900"
    : "bg-gray-100 hover:bg-gray-200 active:bg-gray-300 border-gray-100 hover:border-gray-200 active:border-gray-300 text-slate-900 hover:text-slate-900";
  return (
    <div className={className}>
      <a
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        draggable={false}
        className={`${baseClasses} ${variantClasses}`}
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="currentColor"
          className="w-5 h-5 shrink-0"
          aria-hidden="true"
        >
          <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61-.546-1.387-1.333-1.757-1.333-1.757-1.089-.745.084-.729.084-.729 1.205.084 1.84 1.236 1.84 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.418-1.305.762-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23A11.52 11.52 0 0 1 12 6.803c1.02.005 2.047.138 3.006.404 2.29-1.552 3.293-1.23 3.293-1.23.647 1.653.242 2.873.12 3.176.77.84 1.233 1.91 1.233 3.22 0 4.61-2.807 5.625-5.48 5.92.43.372.823 1.102.823 2.222 0 1.606-.015 2.898-.015 3.293 0 .322.216.694.825.576C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
        </svg>
        {label}
      </a>
    </div>
  );
}

export function Button({ name, to, className, primary = false }: ButtonProps) {
  const baseClasses =
    "rounded-md cursor-pointer text-xl font-bold py-2 px-8 text-center duration-200 transition-colors select-none whitespace-nowrap no-underline hover:no-underline block w-full border border-solid";

  const variantClasses = primary
    ? "bg-magenta hover:bg-magenta-hover active:bg-magenta-active border-magenta hover:border-magenta-hover active:border-magenta-active text-white hover:text-white dark:bg-gray-100 dark:hover:bg-gray-200 dark:active:bg-gray-300 dark:border-gray-100 dark:hover:border-gray-200 dark:text-slate-900 dark:hover:text-slate-900"
    : "bg-gray-100 hover:bg-gray-200 active:bg-gray-300 border-gray-100 hover:border-gray-200 active:border-gray-300 text-slate-900 hover:text-slate-900";

  return (
    <div className={className}>
      <Link
        draggable={false}
        className={`${baseClasses} ${variantClasses}`}
        to={to}
      >
        {name}
      </Link>
    </div>
  );
}
