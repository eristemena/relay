import clsx from "clsx";
import type { ComponentPropsWithoutRef } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

interface FormattedMarkdownProps {
  className?: string;
  content: string;
}

export function FormattedMarkdown({
  className,
  content,
}: FormattedMarkdownProps) {
  return (
    <div className={clsx("relay-markdown", className)}>
      <ReactMarkdown
        components={{
          a: MarkdownLink,
          code: MarkdownCode,
          pre: MarkdownPre,
        }}
        remarkPlugins={[remarkGfm]}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}

function MarkdownLink(props: ComponentPropsWithoutRef<"a">) {
  return <a {...props} className={clsx("relay-markdown-link", props.className)} />;
}

function MarkdownCode({
  className,
  children,
  ...props
}: ComponentPropsWithoutRef<"code"> & { inline?: boolean }) {
  const hasBlockClass = typeof className === "string" && className.length > 0;

  if (hasBlockClass) {
    return (
      <code {...props} className={clsx("relay-markdown-code-block", className)}>
        {children}
      </code>
    );
  }

  return (
    <code {...props} className={clsx("relay-markdown-code-inline", className)}>
      {children}
    </code>
  );
}

function MarkdownPre(props: ComponentPropsWithoutRef<"pre">) {
  return <pre {...props} className={clsx("relay-markdown-pre", props.className)} />;
}