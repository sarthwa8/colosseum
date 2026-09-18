import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { MotionConfig } from 'motion/react'
import './index.css'
import App from './App.tsx'

// reducedMotion="user" is required, not optional: the @media block in index.css
// only governs CSS animations and transitions, while Motion drives inline
// transforms from JS and defaults to reducedMotion:"never". Without this every
// motion.* animation — including a full-screen flash — ignores the OS setting.
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <MotionConfig reducedMotion="user">
      <App />
    </MotionConfig>
  </StrictMode>,
)
