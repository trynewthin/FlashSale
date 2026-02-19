import { cn } from "@/lib/utils"
import { Check } from "lucide-react"

export interface StepperStep {
    label: string
    description?: string
}

interface StepperProps {
    steps: StepperStep[]
    currentStep: number // 0-indexed, -1 means none active
    className?: string
}

export function Stepper({ steps, currentStep, className }: StepperProps) {
    return (
        <div className={cn("flex items-start", className)}>
            {steps.map((step, index) => {
                const isCompleted = index < currentStep
                const isCurrent = index === currentStep
                const isLast = index === steps.length - 1

                return (
                    <div key={index} className={cn("flex items-start", !isLast && "flex-1")}>
                        {/* 步骤圆点 + 文字 */}
                        <div className="flex flex-col items-center gap-1.5">
                            <div
                                className={cn(
                                    "flex size-8 items-center justify-center rounded-full border-2 text-xs font-semibold transition-all duration-300",
                                    isCompleted && "border-emerald-500 bg-emerald-500 text-white",
                                    isCurrent && "border-primary bg-primary text-primary-foreground scale-110 shadow-md shadow-primary/25",
                                    !isCompleted && !isCurrent && "border-muted-foreground/30 text-muted-foreground/50"
                                )}
                            >
                                {isCompleted ? <Check className="size-4" /> : index + 1}
                            </div>
                            <div className="flex flex-col items-center text-center max-w-[80px]">
                                <span
                                    className={cn(
                                        "text-xs font-medium leading-tight",
                                        isCompleted && "text-emerald-600 dark:text-emerald-400",
                                        isCurrent && "text-foreground font-semibold",
                                        !isCompleted && !isCurrent && "text-muted-foreground/60"
                                    )}
                                >
                                    {step.label}
                                </span>
                                {step.description && (
                                    <span className="text-[10px] text-muted-foreground/50 mt-0.5 leading-tight">
                                        {step.description}
                                    </span>
                                )}
                            </div>
                        </div>

                        {/* 连接线 */}
                        {!isLast && (
                            <div className="flex-1 flex items-center pt-4 px-2">
                                <div
                                    className={cn(
                                        "h-0.5 w-full rounded-full transition-all duration-500",
                                        isCompleted ? "bg-emerald-500" : "bg-muted-foreground/15"
                                    )}
                                />
                            </div>
                        )}
                    </div>
                )
            })}
        </div>
    )
}
