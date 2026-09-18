import { useEffect, useRef } from 'react'
import { Camera, Geometry, Mesh, Program, Renderer } from 'ogl'

/**
 * Ambient arena dust. Adapted from the React Bits `Particles` background
 * (reactbits.dev, MIT) — rewritten against ogl 1.0.x with a seeded field and
 * the Colosseum palette.
 *
 * ogl rather than three.js on purpose: ~50KB gzipped vs ~600KB for
 * three + fiber + drei, and a decorative point field needs none of what the
 * larger stack provides. One draw call, gl.POINTS, no geometry.
 *
 * Degrades to nothing when WebGL is unavailable or the user asked for reduced
 * motion — the arena still reads fine without it.
 */

const vertex = /* glsl */ `
  attribute vec3 position;
  attribute vec4 random;
  attribute vec3 color;

  uniform mat4 modelMatrix;
  uniform mat4 viewMatrix;
  uniform mat4 projectionMatrix;
  uniform float uTime;
  uniform float uSpread;
  uniform float uBaseSize;
  uniform float uSizeRandomness;
  uniform vec2 uMouse;

  varying vec4 vRandom;
  varying vec3 vColor;

  void main() {
    vRandom = random;
    vColor = color;

    vec3 pos = position * uSpread;
    pos.z *= 10.0;

    vec4 mPos = modelMatrix * vec4(pos, 1.0);
    float t = uTime;

    // Lazy drift — each particle on its own phase so the field never pulses.
    mPos.x += sin(t * random.z + 6.28 * random.w) * mix(0.1, 1.4, random.x);
    mPos.y += sin(t * random.y + 6.28 * random.x) * mix(0.1, 1.4, random.w);
    mPos.z += sin(t * random.w + 6.28 * random.y) * mix(0.1, 1.4, random.z);

    // Gentle parallax toward the pointer.
    mPos.xy += uMouse * mix(0.25, 1.1, random.x);

    vec4 mvPos = viewMatrix * mPos;
    gl_PointSize = (uBaseSize * (1.0 + uSizeRandomness * (random.x - 0.5))) / length(mvPos.xyz);
    gl_Position = projectionMatrix * mvPos;
  }
`

const fragment = /* glsl */ `
  precision highp float;

  uniform float uTime;
  uniform float uAlpha;

  varying vec4 vRandom;
  varying vec3 vColor;

  void main() {
    // Soft round sprite from point coords — no texture needed.
    vec2 uv = gl_PointCoord.xy;
    float d = length(uv - vec2(0.5));
    if (d > 0.5) discard;

    float edge = smoothstep(0.5, 0.12, d);
    // Slow twinkle, phase-shifted per particle.
    float twinkle = 0.65 + 0.35 * sin(uTime * (0.4 + vRandom.y * 0.8) + vRandom.z * 6.28);

    gl_FragColor = vec4(vColor, edge * uAlpha * twinkle);
  }
`

/** Deterministic PRNG (mulberry32) so the field is identical on every load. */
function seeded(seed: number) {
  let a = seed >>> 0
  return () => {
    a = (a + 0x6d2b79f5) >>> 0
    let t = a
    t = Math.imul(t ^ (t >>> 15), t | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

const PALETTE: [number, number, number][] = [
  [0.941, 0.698, 0.247], // gold
  [0.227, 0.831, 0.812], // fighter A cyan
  [0.956, 0.337, 0.561], // fighter B rose
  [0.55, 0.53, 0.62], // dust
  [0.55, 0.53, 0.62],
]

export interface ParticlesProps {
  count?: number
  spread?: number
  baseSize?: number
  alpha?: number
  className?: string
}

export default function Particles({
  count = 320,
  spread = 11,
  baseSize = 62,
  alpha = 0.5,
  className = '',
}: ParticlesProps) {
  const host = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = host.current
    if (!el) return
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return

    let renderer: Renderer
    try {
      renderer = new Renderer({ depth: false, alpha: true, dpr: Math.min(window.devicePixelRatio, 2) })
    } catch {
      return // no WebGL — the arena is fine without dust
    }

    const gl = renderer.gl
    gl.clearColor(0, 0, 0, 0)
    el.appendChild(gl.canvas)

    const camera = new Camera(gl, { fov: 15 })
    camera.position.set(0, 0, 20)

    const resize = () => {
      const { clientWidth: w, clientHeight: h } = el
      if (!w || !h) return
      renderer.setSize(w, h)
      camera.perspective({ aspect: w / h })
    }
    window.addEventListener('resize', resize, { passive: true })
    resize()

    const mouse = { x: 0, y: 0 }
    const target = { x: 0, y: 0 }
    const onMove = (e: MouseEvent) => {
      const r = el.getBoundingClientRect()
      target.x = ((e.clientX - r.left) / r.width - 0.5) * 2
      target.y = -((e.clientY - r.top) / r.height - 0.5) * 2
    }
    window.addEventListener('mousemove', onMove, { passive: true })

    // Build the field. Rejection-sample into a sphere so density is even
    // rather than clumping at the cube corners.
    const rnd = seeded(0xc01055)
    const positions = new Float32Array(count * 3)
    const randoms = new Float32Array(count * 4)
    const colors = new Float32Array(count * 3)

    for (let i = 0; i < count; i++) {
      let x: number, y: number, z: number, len: number
      do {
        x = rnd() * 2 - 1
        y = rnd() * 2 - 1
        z = rnd() * 2 - 1
        len = x * x + y * y + z * z
      } while (len > 1 || len === 0)
      const r = Math.cbrt(rnd())
      positions.set([x * r, y * r, z * r], i * 3)
      randoms.set([rnd(), rnd(), rnd(), rnd()], i * 4)
      colors.set(PALETTE[Math.floor(rnd() * PALETTE.length)], i * 3)
    }

    const geometry = new Geometry(gl, {
      position: { size: 3, data: positions },
      random: { size: 4, data: randoms },
      color: { size: 3, data: colors },
    })

    const program = new Program(gl, {
      vertex,
      fragment,
      uniforms: {
        uTime: { value: 0 },
        uSpread: { value: spread },
        uBaseSize: { value: baseSize },
        uSizeRandomness: { value: 1 },
        uAlpha: { value: alpha },
        uMouse: { value: [0, 0] },
      },
      transparent: true,
      depthTest: false,
    })

    const points = new Mesh(gl, { mode: gl.POINTS, geometry, program })

    let raf = 0
    let last = performance.now()
    let elapsed = 0

    const frame = (now: number) => {
      raf = requestAnimationFrame(frame)
      const dt = Math.min(now - last, 64) // clamp so tab-switches don't jump
      last = now
      elapsed += dt

      // Ease the pointer so parallax feels weighted, not twitchy.
      mouse.x += (target.x - mouse.x) * 0.045
      mouse.y += (target.y - mouse.y) * 0.045

      program.uniforms.uTime.value = elapsed * 0.00045
      program.uniforms.uMouse.value = [mouse.x, mouse.y]
      points.rotation.y = elapsed * 0.000035
      renderer.render({ scene: points, camera })
    }
    raf = requestAnimationFrame(frame)

    return () => {
      cancelAnimationFrame(raf)
      window.removeEventListener('resize', resize)
      window.removeEventListener('mousemove', onMove)
      gl.canvas.remove()
      const ext = gl.getExtension('WEBGL_lose_context')
      ext?.loseContext()
    }
  }, [count, spread, baseSize, alpha])

  return <div ref={host} className={className} aria-hidden="true" />
}
