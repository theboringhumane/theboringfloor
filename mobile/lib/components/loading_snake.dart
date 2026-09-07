import 'dart:math' as math;

import 'package:flutter/material.dart';

/// A compact, looping progress segment for a row that is currently running.
class LoadingSnake extends StatefulWidget {
  const LoadingSnake({super.key, this.height, this.color, this.trackColor});

  /// The track height. When omitted, this follows the app divider thickness.
  final double? height;

  /// The travelling segment color. Defaults to the theme primary color.
  final Color? color;

  /// The track color. Defaults to the theme elevated surface color.
  final Color? trackColor;

  @override
  State<LoadingSnake> createState() => _LoadingSnakeState();
}

class _LoadingSnakeState extends State<LoadingSnake>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1200),
    );
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (MediaQuery.maybeOf(context)?.disableAnimations ?? false) {
      _controller.stop();
    } else if (!_controller.isAnimating) {
      _controller.repeat();
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final animationDisabled =
        MediaQuery.maybeOf(context)?.disableAnimations ?? false;
    final height = widget.height ?? (theme.dividerTheme.thickness ?? 1) * 2;

    return Semantics(
      label: 'Working',
      child: SizedBox(
        height: height,
        width: double.infinity,
        child: AnimatedBuilder(
          animation: _controller,
          builder: (context, child) => CustomPaint(
            painter: _LoadingSnakePainter(
              progress: animationDisabled ? 0.5 : _controller.value,
              color: widget.color ?? theme.colorScheme.primary,
              trackColor:
                  widget.trackColor ??
                  theme.colorScheme.surfaceContainerHighest,
            ),
          ),
        ),
      ),
    );
  }
}

class _LoadingSnakePainter extends CustomPainter {
  const _LoadingSnakePainter({
    required this.progress,
    required this.color,
    required this.trackColor,
  });

  final double progress;
  final Color color;
  final Color trackColor;

  @override
  void paint(Canvas canvas, Size size) {
    if (size.isEmpty) {
      return;
    }

    final radius = Radius.circular(size.height / 2);
    final track = RRect.fromRectAndRadius(Offset.zero & size, radius);
    canvas.drawRRect(track, Paint()..color = trackColor);

    final segmentWidth = math.min(size.width, size.width * 0.3);
    final travel = size.width + segmentWidth;
    final left = -segmentWidth + (travel * progress);
    final segment = Rect.fromLTWH(left, 0, segmentWidth, size.height);
    final clipped = segment.intersect(Offset.zero & size);
    if (clipped.isEmpty) {
      return;
    }
    canvas.drawRRect(
      RRect.fromRectAndRadius(clipped, radius),
      Paint()..color = color,
    );
  }

  @override
  bool shouldRepaint(_LoadingSnakePainter oldDelegate) =>
      progress != oldDelegate.progress ||
      color != oldDelegate.color ||
      trackColor != oldDelegate.trackColor;
}
