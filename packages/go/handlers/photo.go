// Author: Caden Lund
package handlers

import "bytes"

// MaxPhotoBytes is the most a photo may weigh, for a scan or a plate, as scan.v1's Photo and
// cook.v1's PlateMealRequest state it. The app sends 1024 px at quality 0.7, which lands far
// under it, and app.plate's check holds the same bound: keep the two in step.
const MaxPhotoBytes = 2 << 20

// jpegStart opens every JPEG: the start-of-image marker, then the next marker's first byte.
var jpegStart = []byte{0xFF, 0xD8, 0xFF}

// IsJPEG reports whether photo opens as a JPEG does.
func IsJPEG(photo []byte) bool {
	return bytes.HasPrefix(photo, jpegStart)
}
