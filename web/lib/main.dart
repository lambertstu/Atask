import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'providers/session_provider.dart';
import 'screens/board_screen.dart';

void main() {
  runApp(const AtaskApp());
}

class AtaskApp extends StatelessWidget {
  const AtaskApp({super.key});
  
  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider(
      create: (_) => SessionProvider()..loadProjects()..loadSessions(),
      child: MaterialApp(
        title: 'Atask Board',
        theme: ThemeData(
          colorScheme: ColorScheme.fromSeed(
            seedColor: const Color(0xFF8B5CF6),
            brightness: Brightness.light,
          ),
          useMaterial3: true,
          appBarTheme: const AppBarTheme(
            backgroundColor: Color(0xFFF8F7FA),
            elevation: 0,
            scrolledUnderElevation: 1,
          ),
          scaffoldBackgroundColor: const Color(0xFFF0F1F5),
          cardTheme: CardTheme(
            elevation: 1,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(12),
            ),
          ),
        ),
        home: const BoardScreen(),
      ),
    );
  }
}