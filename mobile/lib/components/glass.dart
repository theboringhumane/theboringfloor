import 'dart:ui';

import 'package:flutter/material.dart';

/// Controls the visual density of a frosted glass surface.
enum GlassIntensity { subtle, regular, strong }

// Subtle glass: a light, background-preserving blur.
const double _subtleBlurSigma = 6;
const double _subtleFillOpacity = 0.08;

// Regular glass: the default floating-control treatment.
const double _regularBlurSigma = 12;
const double _regularFillOpacity = 0.14;

// Strong glass: a more distinct surface for important floating chrome.
const double _strongBlurSigma = 20;
const double _strongFillOpacity = 0.22;

/// A frosted, translucent, rounded surface.
class GlassSurface extends StatelessWidget {
  const GlassSurface({
    super.key,
    required this.child,
    this.padding,
    this.borderRadius,
    this.intensity = GlassIntensity.regular,
  });

  final Widget child;
  final EdgeInsetsGeometry? padding;
  final BorderRadius? borderRadius;
  final GlassIntensity intensity;

  @override
  Widget build(BuildContext context) {
    return _GlassFrame(
      borderRadius: borderRadius ?? BorderRadius.circular(20),
      intensity: intensity,
      child: Padding(
        padding: padding ?? const EdgeInsets.all(16),
        child: child,
      ),
    );
  }
}

/// A fully-rounded frosted pill, for chips and the composer bar.
class GlassPill extends StatelessWidget {
  const GlassPill({
    super.key,
    required this.child,
    this.padding,
    this.onTap,
    this.intensity = GlassIntensity.regular,
  });

  final Widget child;
  final EdgeInsetsGeometry? padding;
  final VoidCallback? onTap;
  final GlassIntensity intensity;

  @override
  Widget build(BuildContext context) {
    const radius = BorderRadius.all(Radius.circular(999));
    final scheme = Theme.of(context).colorScheme;
    final enabled = onTap != null;

    return _GlassFrame(
      borderRadius: radius,
      intensity: intensity,
      child: Material(
        color: scheme.surface.withValues(alpha: 0),
        child: InkWell(
          borderRadius: radius,
          onTap: onTap,
          child: Opacity(
            opacity: enabled ? 1 : 0.5,
            child: Padding(
              padding:
                  padding ??
                  const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              child: child,
            ),
          ),
        ),
      ),
    );
  }
}

/// A circular frosted icon button, for floating top-corner controls.
class GlassIconButton extends StatelessWidget {
  const GlassIconButton({
    super.key,
    required this.icon,
    required this.onPressed,
    this.tooltip,
    this.intensity = GlassIntensity.regular,
  });

  final IconData icon;
  final VoidCallback? onPressed;
  final String? tooltip;
  final GlassIntensity intensity;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final enabled = onPressed != null;
    final button = _GlassFrame(
      borderRadius: const BorderRadius.all(Radius.circular(999)),
      intensity: intensity,
      child: Material(
        color: scheme.surface.withValues(alpha: 0),
        child: InkWell(
          customBorder: const CircleBorder(),
          onTap: onPressed,
          child: SizedBox(
            width: 48,
            height: 48,
            child: Icon(
              icon,
              color: scheme.onSurface.withValues(alpha: enabled ? 0.82 : 0.38),
            ),
          ),
        ),
      ),
    );

    return tooltip == null ? button : Tooltip(message: tooltip!, child: button);
  }
}

class _GlassFrame extends StatelessWidget {
  const _GlassFrame({
    required this.borderRadius,
    required this.intensity,
    required this.child,
  });

  final BorderRadius borderRadius;
  final GlassIntensity intensity;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final fillOpacity = switch (intensity) {
      GlassIntensity.subtle => _subtleFillOpacity,
      GlassIntensity.regular => _regularFillOpacity,
      GlassIntensity.strong => _strongFillOpacity,
    };
    final blurSigma = switch (intensity) {
      GlassIntensity.subtle => _subtleBlurSigma,
      GlassIntensity.regular => _regularBlurSigma,
      GlassIntensity.strong => _strongBlurSigma,
    };

    // ClipRRect bounds the BackdropFilter to this floating surface.
    return ClipRRect(
      borderRadius: borderRadius,
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: blurSigma, sigmaY: blurSigma),
        child: DecoratedBox(
          decoration: BoxDecoration(
            color: scheme.surfaceContainerHighest.withValues(
              alpha: fillOpacity,
            ),
            borderRadius: borderRadius,
            border: Border.all(
              color: scheme.outlineVariant.withValues(alpha: 0.55),
              width: 1,
            ),
          ),
          child: child,
        ),
      ),
    );
  }
}
