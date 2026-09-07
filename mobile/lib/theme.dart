import 'package:flutter/material.dart';
import 'package:theboringfloor/utils/typography.dart';

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
  scaffoldBackgroundColor: colorScheme.surface,
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
    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(20)),
    surfaceTintColor: Colors.transparent,
  ),
  dividerTheme: DividerThemeData(
    color: colorScheme.outlineVariant.withValues(alpha: 0.55),
    thickness: 1,
    space: 1,
  ),
  chipTheme: ChipThemeData(
    backgroundColor: colorScheme.surfaceContainerLow,
    side: BorderSide(color: colorScheme.outlineVariant.withValues(alpha: 0.6)),
    shape: const StadiumBorder(),
  ),
  textTheme: buildAppTextTheme(
    ThemeData(brightness: colorScheme.brightness).textTheme.apply(
      bodyColor: colorScheme.onSurface,
      displayColor: colorScheme.onSurface,
    ),
  ),
  inputDecorationTheme: InputDecorationTheme(
    filled: true,
    fillColor: colorScheme.surfaceContainerLow.withValues(alpha: 0.82),
    contentPadding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
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
