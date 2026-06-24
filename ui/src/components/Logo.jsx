// Inkstone mark: a rounded "stone" slab with an ink well and a single
// ink stroke. Uses currentColor for the stone and the brand accent for
// the ink, so it adapts to light/dark and any rebrand automatically.
export default function Logo({ size = 26, className }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      role="img"
      aria-label="Inkstone logo"
      className={className}
    >
      {/* stone slab */}
      <rect
        x="3.5"
        y="3.5"
        width="25"
        height="25"
        rx="7"
        stroke="currentColor"
        strokeWidth="2.2"
        fill="none"
      />
      {/* ink stroke (diagonal nib) */}
      <path
        d="M10 22L20.5 11.5"
        stroke="var(--brand-accent, #3a6ea5)"
        strokeWidth="2.6"
        strokeLinecap="round"
      />
      {/* ink drop */}
      <circle cx="22" cy="10" r="2.4" fill="var(--brand-accent, #3a6ea5)" />
    </svg>
  );
}
