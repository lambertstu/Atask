import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/session_provider.dart';
import '../models/session.dart';
import '../models/project.dart';
import '../models/message.dart';
import '../widgets/session_card.dart';
import '../widgets/message_bubble.dart';
import '../services/sse.dart';

class BoardScreen extends StatefulWidget {
  const BoardScreen({super.key});

  @override
  State<BoardScreen> createState() => _BoardScreenState();
}

class _BoardScreenState extends State<BoardScreen> {
  final SseService _sseService = SseService();
  final Map<String, StreamSubscription> _sseSubscriptions = {};

  @override
  void dispose() {
    for (var subscription in _sseSubscriptions.values) {
      subscription.cancel();
    }
    _sseService.disconnect();
    super.dispose();
  }

  void _subscribeToSessionEvents(String sessionId, SessionProvider provider) {
    if (_sseSubscriptions.containsKey(sessionId)) return;

    final stream = _sseService.connect(sessionId);
    final subscription = stream.listen((event) {
      provider.handleEvent(event);
    });
    _sseSubscriptions[sessionId] = subscription;
  }

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
          const SizedBox(height: 16),
          _buildProjectList(context, provider),
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
    final colors = {
      null: Colors.grey,
      'Q4 Market Analysis': const Color(0xFF8B5CF6),
      'Competitor Tracking': Colors.blue,
      'Customer Support Automation': Colors.green,
      'Content Strategy': Colors.orange,
    };
    
    final color = colors[title] ?? Colors.grey;
    
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 2),
      decoration: BoxDecoration(
        color: isSelected ? color.withAlpha(38) : Colors.transparent,
        borderRadius: BorderRadius.circular(8),
      ),
      child: ListTile(
        leading: Container(
          width: 8,
          height: 8,
          decoration: BoxDecoration(
            color: color,
            shape: BoxShape.circle,
          ),
        ),
        title: Row(
          children: [
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
        onTap: () => _selectProject(context, project),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
        ),
      ),
    );
  }
  
  void _selectProject(BuildContext context, Project? project) {
    context.read<SessionProvider>().selectProject(project);
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
          _buildActionButton(
            context,
            icon: Icons.folder_open,
            label: '打开项目',
            onPressed: () => _showOpenProjectDialog(context),
            isPrimary: false,
          ),
          const SizedBox(width: 16),
          Expanded(
            child: _buildSearchBar(context),
          ),
          const SizedBox(width: 16),
          _buildActionButton(
            context,
            icon: Icons.add,
            label: 'New Session',
            onPressed: () => _showCreateSessionDialog(context),
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
          const SizedBox(width: 16),
          _buildKanbanColumn(
            context,
            'Completed',
            provider.completedSessions,
            const Color(0xFF059669),
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
                  ? Center(
                      child: Text(
                        'No sessions',
                        style: TextStyle(color: Colors.grey[400]),
                      ),
                    )
                  : ListView.builder(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                      itemCount: sessions.length,
                      itemBuilder: (context, index) {
                        final session = sessions[index];
                        if (session.state == SessionState.processing ||
                            session.state == SessionState.planning) {
                          _subscribeToSessionEvents(session.id, context.read<SessionProvider>());
                        }
                        return Container(
                          margin: const EdgeInsets.only(bottom: 12),
                          child: SessionCard(
                            session: session,
                            onTap: () => _showSessionDetail(context, session),
                            onApprove: session.state == SessionState.blocked
                                ? () => context.read<SessionProvider>().unblock(session.id, true)
                                : null,
                            onReject: session.state == SessionState.blocked
                                ? () => context.read<SessionProvider>().unblock(session.id, false)
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
  
  void _showOpenProjectDialog(BuildContext context) {
    String path = '';
    showDialog(
      context: context,
      builder: (context) {
        return AlertDialog(
          title: const Text('打开项目'),
          content: TextField(
            decoration: const InputDecoration(
              labelText: '项目路径',
              hintText: '输入项目名称或路径',
              border: OutlineInputBorder(),
            ),
            onChanged: (value) => path = value,
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('取消'),
            ),
            ElevatedButton(
              onPressed: () {
                if (path.isNotEmpty) {
                  context.read<SessionProvider>().createProject(path);
                  Navigator.pop(context);
                }
              },
              style: ElevatedButton.styleFrom(
                backgroundColor: const Color(0xFF8B5CF6),
                foregroundColor: Colors.white,
              ),
              child: const Text('打开'),
            ),
          ],
        );
      },
    );
}

void _showCreateSessionDialog(BuildContext context) {
    final provider = context.read<SessionProvider>();
    String? projectPath = provider.selectedProject?.path;
    String input = '';
    showDialog(
      context: context,
      builder: (context) {
        return AlertDialog(
          title: const Text('Create New Session'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (provider.projects.isEmpty)
                  TextField(
                    decoration: const InputDecoration(
                      labelText: 'Project Path',
                      hintText: '/path/to/project',
                      border: OutlineInputBorder(),
                    ),
                    onChanged: (value) => projectPath = value,
                  ),
                if (provider.projects.isNotEmpty)
                  DropdownButtonFormField<String>(
                    value: projectPath,
                    decoration: const InputDecoration(
                      labelText: 'Project',
                      border: OutlineInputBorder(),
                    ),
                    items: provider.projects.map((p) => DropdownMenuItem(
                      value: p.path,
                      child: Text(p.name),
                    )).toList(),
                    onChanged: (value) => projectPath = value,
                  ),
                const SizedBox(height: 16),
                TextField(
                  decoration: const InputDecoration(
                    labelText: 'Task Description',
                    hintText: 'What do you want the agent to do?',
                    border: OutlineInputBorder(),
                    alignLabelWithHint: true,
                  ),
                  maxLines: 4,
                  onChanged: (value) => input = value,
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Cancel'),
            ),
            ElevatedButton(
              onPressed: () async {
                if (projectPath != null && projectPath!.isNotEmpty) {
                  final sess = await provider.createSession(projectPath!);
                  if (input.isNotEmpty) {
                    provider.submitInput(sess.id, input);
                  }
                  if (context.mounted) {
                    Navigator.pop(context);
                  }
                }
              },
              style: ElevatedButton.styleFrom(
                backgroundColor: const Color(0xFF8B5CF6),
                foregroundColor: Colors.white,
              ),
              child: const Text('Create Session'),
            ),
          ],
        );
      },
    );
  }
  
  void _showSessionDetail(BuildContext context, Session session) {
    final processedMessages = _processMessages(session.messages);
    showDialog(
      context: context,
      builder: (context) {
        return AlertDialog(
          title: Row(
            children: [
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: const Color(0xFF8B5CF6).withAlpha(38),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: const Icon(
                  Icons.chat,
                  color: Color(0xFF8B5CF6),
                  size: 24,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Session ${session.id.substring(0, 8)}',
                      style: const TextStyle(fontSize: 16),
                    ),
                    Text(
                      session.projectName,
                      style: TextStyle(
                        fontSize: 12,
                        color: Colors.grey[600],
                      ),
                    ),
                  ],
                ),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                decoration: BoxDecoration(
                  color: _getStateColor(session.state).withAlpha(38),
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Text(
                  session.state.name,
                  style: TextStyle(
                    color: _getStateColor(session.state),
                    fontWeight: FontWeight.bold,
                    fontSize: 12,
                  ),
                ),
              ),
            ],
          ),
          content: SizedBox(
            width: 600,
            height: 500,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: Row(
                    children: [
                      Icon(Icons.model_training, size: 16, color: Colors.grey[600]),
                      const SizedBox(width: 4),
                      Text(
                        session.model,
                        style: TextStyle(
                          fontSize: 12,
                          color: Colors.grey[600],
                        ),
                      ),
                      const SizedBox(width: 12),
                      Icon(Icons.settings, size: 16, color: Colors.grey[600]),
                      const SizedBox(width: 4),
                      Text(
                        session.mode.name,
                        style: TextStyle(
                          fontSize: 12,
                          color: Colors.grey[600],
                        ),
                      ),
                    ],
                  ),
                ),
                const Divider(height: 1),
                Expanded(
                  child: processedMessages.isEmpty
                      ? Center(
                          child: Text(
                            'No messages',
                            style: TextStyle(color: Colors.grey[400]),
                          ),
                        )
                      : ListView.builder(
                          padding: const EdgeInsets.symmetric(vertical: 12),
                          itemCount: processedMessages.length,
                          itemBuilder: (context, index) {
                            final item = processedMessages[index];
                            return MessageBubble(
                              message: item.message,
                              toolResults: item.toolResults,
                            );
                          },
                        ),
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Close'),
            ),
          ],
        );
      },
    );
  }

  List<_ProcessedMessage> _processMessages(List<ChatMessage> messages) {
    final toolResultsMap = <String, String>{};
    
    for (final msg in messages) {
      if (msg.role == 'tool' && msg.toolCallId != null && msg.content != null) {
        toolResultsMap[msg.toolCallId!] = msg.content!;
      }
    }
    
    return messages
        .where((msg) => msg.role != 'tool')
        .map((msg) => _ProcessedMessage(
          message: msg,
          toolResults: toolResultsMap,
        ))
        .toList();
  }

  Color _getStateColor(SessionState state) {
    switch (state) {
      case SessionState.pending:
        return const Color(0xFF3B82F6);
      case SessionState.planning:
        return const Color(0xFF8B5CF6);
      case SessionState.processing:
        return const Color(0xFF10B981);
      case SessionState.blocked:
        return const Color(0xFFF59E0B);
      case SessionState.completed:
        return const Color(0xFF059669);
    }
  }
}

class _ProcessedMessage {
  final ChatMessage message;
  final Map<String, String> toolResults;

  _ProcessedMessage({
    required this.message,
    required this.toolResults,
  });
}