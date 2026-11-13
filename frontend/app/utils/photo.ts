export const PHOTO_MAX_SIDE = 1600
export const PHOTO_QUALITY = 0.8

/**
 * Decodes an image file, scales its long side down to PHOTO_MAX_SIDE and re-encodes it as JPEG.
 * Only pixels are written, so EXIF (GPS included) and any other hidden data are dropped.
 * Rejects when the file cannot be read as an image.
 */
export async function processPhoto(file: File): Promise<Blob> {
  if (!file.type.startsWith('image/')) throw new Error('Not an image')

  // 'from-image' applies the EXIF rotation to the pixels, so the photo stays upright without it.
  const bitmap = await createImageBitmap(file, { imageOrientation: 'from-image' })
  try {
    const scale = Math.min(1, PHOTO_MAX_SIDE / Math.max(bitmap.width, bitmap.height))
    const canvas = document.createElement('canvas')
    canvas.width = Math.max(1, Math.round(bitmap.width * scale))
    canvas.height = Math.max(1, Math.round(bitmap.height * scale))
    const context = canvas.getContext('2d')
    if (!context) throw new Error('Canvas unavailable')
    context.drawImage(bitmap, 0, 0, canvas.width, canvas.height)

    return await new Promise<Blob>((resolve, reject) => {
      canvas.toBlob(blob => (blob ? resolve(blob) : reject(new Error('Could not encode the photo'))), 'image/jpeg', PHOTO_QUALITY)
    })
  } finally {
    bitmap.close()
  }
}
