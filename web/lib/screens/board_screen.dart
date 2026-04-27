import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:file_picker/file_picker.dart';
import '../providers/session_provider.dart';
import '../providers/settings_provider.dart';
import '../models/session.dart';
import '../models/project.dart';
import '../widgets/session_card.dart';
import '../widgets/session_detail_dialog.dart';
import '../widgets/llm_config_dialog.dart';

class BoardScreen extends StatelessWidget {
  const BoardScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Consumer<SessionProvider>(
        builder: (context, provider, _) {
          if (provider.isLoading || provider.isLoadingProjects) {
            return const Center(child: CircularProgressIndicator());
          }

          if (provider.error != null) {
            return Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  const Icon(Icons.error, size: 48, color: Colors.red),
                  const SizedBox(height: 16),
                  Text(provider.error!),
                  const SizedBox(height: 16),
                  ElevatedButton(
                    onPressed: () {
                      provider.clearError();
                      provider.loadProjects();
                      provider.loadSessions();
                    },
                    child: const Text('Retry'),
                  ),
                ],
              ),
            );
          }

          return Row(
            children: [
              _buildSidebar(context, provider),
              Expanded(
                child: Column(
                  children: [
                    _buildTopBar(context, provider),
                    Expanded(
                      child: _buildKanbanBoard(context, provider),
                    ),
                  ],
                ),
              ),
            ],
          );
        },
      ),
    );
  }

  Widget _buildSidebar(BuildContext context, SessionProvider provider) {
    return Container(
      width: 260,
      decoration: BoxDecoration(
        color: const Color(0xFFF8F7FA),
        border: Border(right: BorderSide(color: Colors.grey[300]!)),
      ),
      child: Column(
        children: [
          _buildLogo(context),
          const SizedBox(height: 8),
          _buildOpenProjectButton(context),
          const SizedBox(height: 16),
          _buildProjectList(context, provider),
          _buildSettingsButton(context),
        ],
      ),
    );
  }

  Widget _buildLogo(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(20),
      child: Row(
        children: [
          Image.asset(
            'lib/assets/logo.png',
            height: 40,
            width: 40,
          ),
          const SizedBox(width: 12),
          const Text(
            'Atask',
            style: TextStyle(
              fontSize: 22,
              fontWeight: FontWeight.bold,
              color: Color(0xFF8B5CF6),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildProjectList(BuildContext context, SessionProvider provider) {
    return Expanded(
      child: ListView(
        padding: const EdgeInsets.symmetric(horizontal: 12),
        children: [
          ...provider.projects.map((project) => _buildProjectItem(
            context,
            project,
            project.name,
            provider.selectedProject?.path == project.path,
          )),
        ],
      ),
    );
  }

  Widget _buildProjectItem(
    BuildContext context,
    Project? project,
    String title,
    bool isSelected,
  ) {
    final color = project != null ? _getProjectColor(project.name) : Colors.grey;

    return Container(
      margin: const EdgeInsets.symmetric(vertical: 2),
      decoration: BoxDecoration(
        color: isSelected ? color.withAlpha(38) : Colors.transparent,
        borderRadius: BorderRadius.circular(8),
      ),
      child: InkWell(
        onTap: () => context.read<SessionProvider>().selectProject(project),
        borderRadius: BorderRadius.circular(8),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          child: Row(
            children: [
              Container(
                width: 4,
                height: 32,
                decoration: BoxDecoration(
                  color: color,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  title,
                  style: TextStyle(
                    fontWeight: isSelected ? FontWeight.bold : FontWeight.normal,
                    color: isSelected ? color : Colors.grey[800],
                    fontSize: 14,
                  ),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              if (isSelected)
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                  decoration: BoxDecoration(
                    color: color.withAlpha(38),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(
                    'Active',
                    style: TextStyle(
                      color: color,
                      fontWeight: FontWeight.bold,
                      fontSize: 12,
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildOpenProjectButton(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: OutlinedButton.icon(
        icon: const Icon(Icons.folder_open, size: 18),
        label: const Text('open project'),
        style: OutlinedButton.styleFrom(
          foregroundColor: const Color(0xFF8B5CF6),
          side: const BorderSide(color: Color(0xFFE5E7EB)),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
          ),
        ),
        onPressed: () => _showOpenProjectDialog(context),
      ),
    );
  }

  Widget _buildTopBar(BuildContext context, SessionProvider provider) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
      decoration: BoxDecoration(
        color: const Color(0xFFF8F7FA),
        border: Border(bottom: BorderSide(color: Colors.grey[300]!)),
      ),
      child: Row(
        children: [
          Expanded(
            child: _buildSearchBar(context),
          ),
          const SizedBox(width: 16),
          _buildActionButton(
            context,
            icon: Icons.add,
            label: 'New Session',
            onPressed: () => _createSessionDirectly(context),
            isPrimary: true,
          ),
          const SizedBox(width: 8),
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () {
              provider.loadProjects();
              provider.loadSessions(projectName: provider.selectedProject?.name);
            },
            tooltip: 'Refresh',
          ),
        ],
      ),
    );
  }

  Widget _buildSearchBar(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: Colors.grey[300]!),
      ),
      child: const TextField(
        decoration: InputDecoration(
          hintText: 'Search agents or tasks...',
          prefixIcon: Icon(Icons.search, color: Colors.grey),
          border: InputBorder.none,
          contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        ),
      ),
    );
  }

  Widget _buildActionButton(
    BuildContext context, {
    required IconData icon,
    required String label,
    required VoidCallback? onPressed,
    bool isPrimary = false,
  }) {
    if (isPrimary) {
      return ElevatedButton.icon(
        icon: Icon(icon, size: 18),
        label: Text(label),
        style: ElevatedButton.styleFrom(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
          ),
        ),
        onPressed: onPressed,
      );
    }

    return OutlinedButton.icon(
      icon: Icon(icon, size: 18),
      label: Text(label),
      style: OutlinedButton.styleFrom(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
        ),
      ),
      onPressed: onPressed,
    );
  }

  Widget _buildKanbanBoard(BuildContext context, SessionProvider provider) {
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _buildKanbanColumn(
            context,
            'Completed',
            provider.completedSessions,
            const Color(0xFF059669),
          ),
          const SizedBox(width: 16),
          _buildKanbanColumn(
            context,
            'Scheduled',
            provider.pendingSessions,
            Colors.blue,
          ),
          const SizedBox(width: 16),
          _buildKanbanColumn(
            context,
            'Planning',
            provider.planningSessions,
            const Color(0xFF8B5CF6),
          ),
          const SizedBox(width: 16),
          _buildKanbanColumn(
            context,
            'In Progress',
            provider.processingSessions,
            const Color(0xFF10B981),
          ),
          const SizedBox(width: 16),
          _buildKanbanColumn(
            context,
            'Human Review',
            provider.blockedSessions,
            Colors.orange,
          ),
        ],
      ),
    );
  }

  Widget _buildKanbanColumn(
    BuildContext context,
    String title,
    List<Session> sessions,
    Color headerColor,
  ) {
    return Expanded(
      child: Container(
        decoration: BoxDecoration(
          color: const Color(0xFFF0F1F5),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(
            color: Colors.grey[300]!,
            width: 1,
          ),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.all(12),
              child: Row(
                children: [
                  Text(
                    title,
                    style: TextStyle(
                      fontWeight: FontWeight.bold,
                      fontSize: 14,
                      color: headerColor,
                    ),
                  ),
                  const SizedBox(width: 8),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: headerColor.withAlpha(38),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Text(
                      '${sessions.length}',
                      style: TextStyle(
                        color: headerColor,
                        fontWeight: FontWeight.bold,
                        fontSize: 12,
                      ),
                    ),
                  ),
                ],
              ),
            ),
            Expanded(
              child: sessions.isEmpty
                  ? const SizedBox.shrink()
                  : ListView.builder(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                      itemCount: sessions.length,
                      itemBuilder: (context, index) {
                        final session = sessions[index];
                        return Container(
                          margin: const EdgeInsets.only(bottom: 12),
                          child: SessionCard(
                            session: session,
                            onTap: () => _showSessionDetail(context, session),
                            onApprove: session.state == SessionState.blocked
                                ? (addAllowed) => context.read<SessionProvider>().unblock(session.id, true, addAllowed: addAllowed)
                                : null,
                            onReject: session.state == SessionState.blocked
                                ? (addAllowed) => context.read<SessionProvider>().unblock(session.id, false, addAllowed: addAllowed)
                                : null,
                          ),
                        );
                      },
                    ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _showOpenProjectDialog(BuildContext context) async {
    final selectedPath = await FilePicker.platform.getDirectoryPath(
      dialogTitle: '选择项目目录',
    );

    if (selectedPath == null) return;

    if (!context.mounted) return;

    try {
      await context.read<SessionProvider>().createProject(selectedPath);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('项目已打开: ${selectedPath.split('/').last}')),
        );
      }
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('打开项目失败: $e')),
        );
      }
    }
  }

  Future<void> _createSessionDirectly(BuildContext context) async {
    final provider = context.read<SessionProvider>();
    final settingsProvider = context.read<SettingsProvider>();

    if (provider.selectedProject == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请先选择一个项目')),
      );
      return;
    }

    try {
      final session = await provider.createSession(
        provider.selectedProject!.path,
        model: settingsProvider.config?.model ?? 'glm-5',
      );

      provider.loadSessions(projectName: provider.selectedProject?.name);

      if (context.mounted) {
        _showSessionDetail(context, session);
      }
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('创建会话失败: $e')),
        );
      }
    }
  }

  void _showSessionDetail(BuildContext context, Session session) {
    showDialog(
      context: context,
      builder: (context) {
        return SessionDetailDialog(sessionId: session.id);
      },
    );
  }

  Widget _buildSettingsButton(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      child: InkWell(
        onTap: () => _showLLMConfigDialog(context),
        borderRadius: BorderRadius.circular(8),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          child: Row(
            children: [
              Icon(Icons.settings, size: 20, color: Colors.grey[600]),
              const SizedBox(width: 12),
              Text(
                'Settings',
                style: TextStyle(color: Colors.grey[700], fontSize: 14),
              ),
            ],
          ),
        ),
      ),
    );
  }

  void _showLLMConfigDialog(BuildContext context) {
    showDialog(
      context: context,
      builder: (context) {
        return const LLMConfigDialog();
      },
    );
  }

  Color _getProjectColor(String projectName) {
    final colors = [
      const Color(0xFF8B5CF6),
      const Color(0xFF3B82F6),
      const Color(0xFF10B981),
      const Color(0xFFF59E0B),
      const Color(0xFFEF4444),
      const Color(0xFF14B8A6),
      const Color(0xFFEC4899),
      const Color(0xFF6366F1),
    ];
    final hash = projectName.hashCode;
    return colors[hash.abs() % colors.length];
  }
}