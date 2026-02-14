import { Button } from "@/components/ui/button"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { MoreHorizontal } from "lucide-react"

import { type OrderView } from "@/api/modules/order"
import { canReviewOrder, canShipOrder } from "@/features/order/status"

export function OrderRowActions({ order, canReview, canShip, onReview, onShip }: {
  order: OrderView; canReview: boolean; canShip: boolean; onReview: () => void; onShip: () => void
}) {
  const showReview = canReview && canReviewOrder(order)
  const showShip = canShip && canShipOrder(order)
  if (!showReview && !showShip) return null
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-7" />}>
        <MoreHorizontal className="size-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {showReview && <DropdownMenuItem onClick={onReview}>审核</DropdownMenuItem>}
        {showShip && <DropdownMenuItem onClick={onShip}>发货</DropdownMenuItem>}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
