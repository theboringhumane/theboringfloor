import 'package:flutter/material.dart';

class EmptyState extends StatelessWidget {
  const EmptyState(this.message, {super.key});
  final String message;
  @override
  Widget build(BuildContext context) => Center(
    child: Text(message, style: Theme.of(context).textTheme.bodyMedium),
  );
}
