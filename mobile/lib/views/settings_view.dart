import 'package:flutter/material.dart';

import '../store/settings_store.dart';
import '../utils/typography.dart';

class SettingsView extends StatefulWidget {
  const SettingsView({super.key, required this.store});
  final SettingsStore store;
  @override
  State<SettingsView> createState() => _SettingsViewState();
}

class _SettingsViewState extends State<SettingsView> {
  late final TextEditingController url, token;
  String? status;
  bool tokenVisible = false;
  @override
  void initState() {
    super.initState();
    url = TextEditingController(text: widget.store.settings.baseUrl);
    token = TextEditingController(text: widget.store.settings.token);
  }

  @override
  void dispose() {
    url.dispose();
    token.dispose();
    super.dispose();
  }

  Future<void> _test() async {
    try {
      await widget.store.client().health();
      if (mounted) {
        setState(() => status = 'Connection successful.');
      }
    } catch (_) {
      if (mounted) {
        setState(() => status = 'Connection failed.');
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return SafeArea(
      child: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          Text(
            'Settings',
            style: AppFonts.heading(
              context,
              base: theme.textTheme.headlineLarge,
            ),
          ),
          const SizedBox(height: 20),
          TextField(
            controller: url,
            style: AppFonts.mono(context, base: theme.textTheme.bodyLarge),
            decoration: InputDecoration(
              labelText: 'Gateway base URL',
              labelStyle: AppFonts.body(
                context,
                base: theme.textTheme.bodyLarge,
              ),
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: token,
            obscureText: !tokenVisible,
            style: AppFonts.mono(context, base: theme.textTheme.bodyLarge),
            decoration: InputDecoration(
              labelText: 'Bearer token',
              labelStyle: AppFonts.body(
                context,
                base: theme.textTheme.bodyLarge,
              ),
              suffixIcon: IconButton(
                tooltip: tokenVisible ? 'Hide token' : 'Show token',
                onPressed: () => setState(() => tokenVisible = !tokenVisible),
                icon: Icon(
                  tokenVisible
                      ? Icons.visibility_off_outlined
                      : Icons.visibility,
                ),
              ),
            ),
          ),
          const SizedBox(height: 16),
          Wrap(
            spacing: 8,
            children: [
              FilledButton(
                onPressed: () => widget.store.save(
                  GatewaySettings(baseUrl: url.text, token: token.text),
                ),
                child: const Text('Save'),
              ),
              OutlinedButton(
                onPressed: _test,
                child: const Text('Test connection'),
              ),
            ],
          ),
          if (status != null)
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: Text(status!),
            ),
        ],
      ),
    );
  }
}
