export function BrandMark({ className = "" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 32 32"
      fill="currentColor"
      className={className}
      aria-hidden="true"
    >
      <path d="M3 3h11v6H9v5h5v6H9v9H3V3Zm15 0h11v6H18V3Zm0 11h11v6H18v-6Zm0 9h11v6H18v-6Z" />
    </svg>
  );
}
