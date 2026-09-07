import 'dart:convert';
import 'dart:typed_data';

const int maxAttachmentsPerMessage = 4;
const int maxAttachmentBytes = 8 * 1024 * 1024;
const int maxAttachmentTotalBytes = 16 * 1024 * 1024;

const allowedAttachmentMimeTypes = <String>{
  'image/png',
  'image/jpeg',
  'image/gif',
  'image/webp',
};

class Attachment {
  const Attachment({
    required this.name,
    required this.mimeType,
    required this.bytes,
  });

  final String name;
  final String mimeType;
  final Uint8List bytes;

  int get sizeInBytes => bytes.lengthInBytes;

  Map<String, String> toJson() => {
    'name': name,
    'mimeType': mimeType,
    'data': base64Encode(bytes),
  };
}

class AttachmentValidationException implements Exception {
  const AttachmentValidationException(this.message);
  final String message;
}

class AttachmentValidator {
  const AttachmentValidator._();

  static String? mimeTypeForName(String name) {
    final dot = name.lastIndexOf('.');
    if (dot < 0 || dot == name.length - 1) return null;
    return switch (name.substring(dot + 1).toLowerCase()) {
      'png' => 'image/png',
      'jpg' || 'jpeg' => 'image/jpeg',
      'gif' => 'image/gif',
      'webp' => 'image/webp',
      _ => null,
    };
  }

  static void validate(Iterable<Attachment> attachments) {
    final list = attachments.toList();
    if (list.length > maxAttachmentsPerMessage) {
      throw const AttachmentValidationException(
        'Up to 4 images can be attached.',
      );
    }
    var totalBytes = 0;
    for (final attachment in list) {
      if (!allowedAttachmentMimeTypes.contains(attachment.mimeType)) {
        throw const AttachmentValidationException(
          'Only PNG, JPEG, GIF, and WebP images can be attached.',
        );
      }
      if (attachment.sizeInBytes > maxAttachmentBytes) {
        throw const AttachmentValidationException(
          'Each image must be 8 MiB or smaller.',
        );
      }
      totalBytes += attachment.sizeInBytes;
    }
    if (totalBytes > maxAttachmentTotalBytes) {
      throw const AttachmentValidationException(
        'Attachments must total 16 MiB or smaller.',
      );
    }
  }
}
