import 'package:flutter/material.dart';

class SplashView extends StatelessWidget {
  const SplashView({super.key, this.errorMessage, this.onRetry});

  final String? errorMessage;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return ColoredBox(
      color: colors.surface,
      child: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Image.asset(
                'assets/logo.png',
                width: 112,
                height: 112,
                color: colors.onSurface,
                colorBlendMode: BlendMode.srcIn,
              ),
              const SizedBox(height: 24),
              if (errorMessage case final message?) ...[
                Text(
                  message,
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.bodyLarge
                      ?.copyWith(color: colors.onSurface),
                ),
                const SizedBox(height: 16),
                OutlinedButton(onPressed: onRetry, child: const Text('Retry')),
              ] else
                SizedBox(
                  width: 28,
                  height: 28,
                  child: CircularProgressIndicator(color: colors.primary),
                ),
            ],
          ),
        ),
      ),
    );
  }
}
