import 'package:image_picker/image_picker.dart' as image_picker;

import '../models/attachment.dart';

enum AttachmentPickSource { photoLibrary, camera }

/// Platform boundary for selecting images. Tests can supply a fake without
/// registering an image_picker platform channel.
abstract class AttachmentPicker {
  Future<List<Attachment>> pick(AttachmentPickSource source);
}

class ImagePickerAttachmentPicker implements AttachmentPicker {
  ImagePickerAttachmentPicker({image_picker.ImagePicker? picker})
    : _picker = picker ?? image_picker.ImagePicker();

  final image_picker.ImagePicker _picker;

  @override
  Future<List<Attachment>> pick(AttachmentPickSource source) async {
    try {
      final files = source == AttachmentPickSource.photoLibrary
          ? await _picker.pickMultiImage()
          : await _pickCameraImage();
      final attachments = <Attachment>[];
      for (final file in files) {
        final mimeType = AttachmentValidator.mimeTypeForName(file.name);
        if (mimeType == null) {
          throw const AttachmentValidationException(
            'Only PNG, JPEG, GIF, and WebP images can be attached.',
          );
        }
        attachments.add(
          Attachment(
            name: file.name,
            mimeType: mimeType,
            bytes: await file.readAsBytes(),
          ),
        );
      }
      AttachmentValidator.validate(attachments);
      return attachments;
    } on AttachmentValidationException {
      rethrow;
    } catch (_) {
      // Cancellation, permission denial, and unsupported camera access don't
      // leave a platform exception in the composer.
      return const [];
    }
  }

  Future<List<image_picker.XFile>> _pickCameraImage() async {
    final image = await _picker.pickImage(
      source: image_picker.ImageSource.camera,
    );
    return image == null ? const [] : [image];
  }
}
