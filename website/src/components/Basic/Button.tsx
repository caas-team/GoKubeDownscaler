import Link from "@docusaurus/Link";

type ButtonProps = {
  name: string;
  to: string;
  className?: string;
};

export function Button({ name, to, className }: ButtonProps) {
  return (
    <div className={className}>
      <Link
        draggable={false}
        className="useTailwind bg-gray-100 hover:bg-gray-200 active:bg-gray-300 border border-solid border-gray-100 hover:border-gray-200 active:border-gray-300 rounded-md cursor-pointer text-xl font-bold py-2 px-8 text-center duration-200 transition-colors select-none whitespace-nowrap text-slate-900 hover:text-slate-900 no-underline hover:no-underline block w-full"
        to={to}
      >
        {name}
      </Link>
    </div>
  );
}
