import 'package:flutter/material.dart';
import 'package:theboringfloor/components/glass.dart';
import 'package:theboringfloor/utils/typography.dart';
import 'package:theboringfloor/utils/working_words.dart';

/// A compact team-status pill that rotates its working verb every four seconds.
class WorkingChip extends StatefulWidget {
  const WorkingChip({
    super.key,
    required this.working,
    this.icon = Icons.auto_awesome,
    this.padding,
    this.intensity = GlassIntensity.regular,
  });

  /// Whether the office currently has work in flight.
  final bool working;

  /// The leading working-status icon.
  final IconData icon;

  /// Optional padding forwarded to the frosted pill.
  final EdgeInsetsGeometry? padding;

  /// The glass density used for the pill.
  final GlassIntensity intensity;

  @override
  State<WorkingChip> createState() => _WorkingChipState();
}

class _WorkingChipState extends State<WorkingChip>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  var _tick = 0;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 4),
    )..addStatusListener(_handleStatus);
  }

  @override
  void didUpdateWidget(covariant WorkingChip oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.working != widget.working) {
      _syncAnimation();
    }
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _syncAnimation();
  }

  void _syncAnimation() {
    if (!widget.working ||
        (MediaQuery.maybeOf(context)?.disableAnimations ?? false)) {
      _controller.stop();
    } else if (!_controller.isAnimating) {
      _controller.forward(from: 0);
    }
  }

  void _handleStatus(AnimationStatus status) {
    if (status == AnimationStatus.completed && mounted && widget.working) {
      setState(() => _tick++);
      _controller.forward(from: 0);
    }
  }

  @override
  void dispose() {
    _controller
      ..removeStatusListener(_handleStatus)
      ..dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (!widget.working) {
      return const SizedBox.shrink();
    }

    final scheme = Theme.of(context).colorScheme;
    final word = workingWordAt(_tick);
    final base = Theme.of(context).textTheme.bodySmall;

    return Semantics(
      label: 'Team is working — $word',
      child: GlassPill(
        padding: widget.padding,
        intensity: widget.intensity,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(widget.icon, color: scheme.primary),
            Text(
              ' team is working — ',
              style: AppFonts.body(context, base: base),
            ),
            Text(
              '$word…',
              style: AppFonts.body(
                context,
                base: base?.copyWith(
                  color: scheme.primary,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
