import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog"

import { type AdminProduct } from "@/api/modules/product"
import { useDeleteProductMutation } from "@/hooks/biz/use-product-mgmt-hooks"

export function DeleteProductDialog({ product, onClose }: { product: AdminProduct; onClose: () => void }) {
  const deleteMutation = useDeleteProductMutation(product.product_id)
  return (
    <AlertDialog open onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除商品</AlertDialogTitle>
          <AlertDialogDescription>此操作不可逆，确定要删除商品 "{product.name}" 吗？</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction onClick={() => deleteMutation.mutate(undefined, { onSuccess: onClose })} disabled={deleteMutation.isPending}>
            {deleteMutation.isPending ? "删除中…" : "确认删除"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
