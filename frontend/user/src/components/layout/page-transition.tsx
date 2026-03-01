import { motion } from "motion/react"

export function PageTransition({ children, className }: { children: React.ReactNode, className?: string }) {
    return (
        <motion.div
            initial={{ opacity: 0, scale: 0.98, y: 5 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.98, y: -5 }}
            transition={{
                duration: 0.2,
                ease: "easeOut"
            }}
            className={className}
        >
            {children}
        </motion.div>
    )
}
