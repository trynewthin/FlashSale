import { useEffect, useState } from "react"
import Counter from "@/components/Counter"

interface SeckillCountdownProps {
    endUnix: number
    fontSize?: number
}

function calcRemaining(endUnix: number) {
    const diff = Math.max(0, endUnix * 1000 - Date.now())
    const h = Math.floor(diff / 3600000)
    const m = Math.floor((diff % 3600000) / 60000)
    const s = Math.floor((diff % 60000) / 1000)
    return { h, m, s, ended: diff <= 0 }
}

const counterProps = {
    padding: 2,
    gap: 0,
    horizontalPadding: 0,
    textColor: "rgb(239 68 68)",
    fontWeight: 800 as const,
    gradientFrom: "transparent",
    gradientTo: "transparent",
}

export function SeckillCountdown({ endUnix, fontSize = 42 }: SeckillCountdownProps) {
    const [remaining, setRemaining] = useState(() => calcRemaining(endUnix))

    useEffect(() => {
        const timer = setInterval(() => {
            setRemaining(calcRemaining(endUnix))
        }, 1000)
        return () => clearInterval(timer)
    }, [endUnix])

    if (remaining.ended) {
        return <span className="text-2xl text-muted-foreground font-medium">已结束</span>
    }

    return (
        <span className="inline-flex items-center gap-2 text-muted-foreground">
            <span className="text-2xl font-medium">本会场</span>
            <span className="text-2xl font-medium">剩余</span>
            <Counter value={remaining.h} places={[10, 1]} fontSize={fontSize} {...counterProps} />
            <span className="text-2xl">小时</span>
            <Counter value={remaining.m} places={[10, 1]} fontSize={fontSize} {...counterProps} />
            <span className="text-2xl">分钟</span>
            <Counter value={remaining.s} places={[10, 1]} fontSize={fontSize} {...counterProps} />
            <span className="text-2xl">秒</span>
            <span className="text-2xl font-medium">结束！</span>
        </span>
    )
}
