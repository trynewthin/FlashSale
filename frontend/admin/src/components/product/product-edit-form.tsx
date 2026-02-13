import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"

import { type AdminProduct } from "@/api/modules/product"
import { useUpdateProductMutation } from "@/hooks/biz/use-product-mgmt-hooks"
import { useApiError } from "@/hooks/common/use-api-error"
import { ProductFormFields, type ProductFormData } from "./product-form-fields"

export function ProductEditForm({ product, formData, setFormData, onClose }: {
  product: AdminProduct; formData: ProductFormData; setFormData: (d: ProductFormData) => void; onClose: () => void
}) {
  const updateMutation = useUpdateProductMutation(product.product_id)
  const { toUserMessage } = useApiError()
  return (
    <form onSubmit={(e) => { e.preventDefault(); updateMutation.mutate(formData, { onSuccess: onClose }) }} className="mt-4 space-y-4">
      <ProductFormFields formData={formData} setFormData={setFormData} />
      {updateMutation.isError && <Alert variant="destructive"><AlertDescription>{toUserMessage(updateMutation.error)}</AlertDescription></Alert>}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onClose}>取消</Button>
        <Button type="submit" disabled={updateMutation.isPending}>{updateMutation.isPending ? "保存中…" : "保存"}</Button>
      </div>
    </form>
  )
}
