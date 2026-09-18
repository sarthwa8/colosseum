/**
 * Metallic sweep across text. Adapted from the React Bits `ShinyText` effect
 * (reactbits.dev, MIT) — a masked gradient that travels left to right.
 *
 * Reserved for the wordmark and the winner reveal; used anywhere else it stops
 * reading as special.
 */
export interface ShinyTextProps {
  text: string
  className?: string
  /** seconds per sweep */
  speed?: number
  disabled?: boolean
}

export default function ShinyText({
  text,
  className = '',
  speed = 4,
  disabled = false,
}: ShinyTextProps) {
  if (disabled) return <span className={className}>{text}</span>

  return (
    <span
      className={`bg-clip-text text-transparent ${className}`}
      style={{
        backgroundImage:
          'linear-gradient(110deg, var(--color-gold-deep) 20%, #fff3d6 42%, var(--color-gold) 52%, var(--color-gold-deep) 75%)',
        backgroundSize: '220% 100%',
        animation: `shiny-sweep ${speed}s linear infinite`,
      }}
    >
      {text}
      <style>{`@keyframes shiny-sweep { 0% { background-position: 180% 0; } 100% { background-position: -80% 0; } }`}</style>
    </span>
  )
}
