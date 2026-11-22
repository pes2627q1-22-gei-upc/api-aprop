package s3_client

import "context"

// AvatarField points to a profile picture of a response that has to be resolved in place.
type AvatarField struct {
	UserID  string
	Picture *string
}

// ResolveAvatars replaces each stored profile picture with the URL that clients can load (#247), using a single
// signing request. Pictures that cannot be resolved become empty and the error is returned for the logs; external
// URLs are always kept. With a nil signer only the external URLs are kept.
func ResolveAvatars(ctx context.Context, signer AvatarSigner, fields []AvatarField) error {
	if len(fields) == 0 {
		return nil
	}
	if signer == nil {
		for _, field := range fields {
			if !IsExternalPicture(*field.Picture) {
				*field.Picture = ""
			}
		}
		return nil
	}
	refs := make([]AvatarRef, len(fields))
	for i, field := range fields {
		refs[i] = AvatarRef{UserID: field.UserID, Stored: *field.Picture}
	}
	urls, err := signer.AvatarURLs(ctx, refs)
	for i, field := range fields {
		if i < len(urls) {
			*field.Picture = urls[i]
		} else if !IsExternalPicture(*field.Picture) {
			*field.Picture = ""
		}
	}
	return err
}
