import 'package:flutter/material.dart';

const appSeedColor = Color(0xFF3F5F5B);

ColorScheme seededLightColorScheme() =>
    ColorScheme.fromSeed(seedColor: appSeedColor, brightness: Brightness.light);

ColorScheme seededDarkColorScheme() =>
    ColorScheme.fromSeed(seedColor: appSeedColor, brightness: Brightness.dark);

ThemeData buildLightTheme([ColorScheme? colorScheme]) =>
    _buildTheme(colorScheme ?? seededLightColorScheme());

ThemeData buildDarkTheme([ColorScheme? colorScheme]) =>
    _buildTheme(colorScheme ?? seededDarkColorScheme());

ThemeData _buildTheme(ColorScheme colorScheme) => ThemeData(
  colorScheme: colorScheme,
  useMaterial3: true,
  appBarTheme: AppBarTheme(
    backgroundColor: colorScheme.surface,
    foregroundColor: colorScheme.onSurface,
    elevation: 0,
    scrolledUnderElevation: 0,
    surfaceTintColor: Colors.transparent,
  ),
  cardTheme: CardThemeData(
    color: colorScheme.surfaceContainerLow,
    elevation: 0,
    margin: EdgeInsets.zero,
    surfaceTintColor: Colors.transparent,
  ),
  dividerTheme: DividerThemeData(color: colorScheme.outlineVariant),
  inputDecorationTheme: InputDecorationTheme(
    filled: true,
    fillColor: colorScheme.surfaceContainerLow,
    border: OutlineInputBorder(
      borderSide: BorderSide(color: colorScheme.outlineVariant),
      borderRadius: BorderRadius.circular(16),
    ),
    enabledBorder: OutlineInputBorder(
      borderSide: BorderSide(color: colorScheme.outlineVariant),
      borderRadius: BorderRadius.circular(16),
    ),
    focusedBorder: OutlineInputBorder(
      borderSide: BorderSide(color: colorScheme.primary, width: 2),
      borderRadius: BorderRadius.circular(16),
    ),
  ),
);
