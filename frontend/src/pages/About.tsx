import React, { useEffect, useState } from 'react';
import { Box, Typography, Paper, Link, CircularProgress } from '@mui/material';
import { getVersionInfo, VersionInfo } from '../api/version';

const About: React.FC = () => {
    const [versions, setVersions] = useState<VersionInfo | null>(null);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchVersions = async () => {
            try {
                const data = await getVersionInfo();
                setVersions(data);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Failed to fetch version information');
            }
        };

        fetchVersions();
    }, []);

    if (error) {
        return (
            <Box sx={{ p: 3 }}>
                <Typography color="error">Error: {error}</Typography>
            </Box>
        );
    }

    if (!versions) {
        return (
            <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
                <CircularProgress />
            </Box>
        );
    }

    return (
        <Box sx={{ p: 3 }}>
            <Typography variant="h4" gutterBottom>
                About KrakenHashes
            </Typography>

            <Paper sx={{ p: 3, mb: 3 }}>
                <Typography variant="h6" gutterBottom>
                    Version Information
                </Typography>
                <Box sx={{ display: 'grid', gridTemplateColumns: 'auto 1fr', gap: 2 }}>
                    {versions.release && (
                        <>
                            <Typography><strong>Release:</strong></Typography>
                            <Typography sx={{ fontWeight: 600 }}>{versions.release}</Typography>
                        </>
                    )}

                    <Typography><strong>Backend:</strong></Typography>
                    <Typography>{versions.backend}</Typography>

                    <Typography><strong>Frontend:</strong></Typography>
                    <Typography>{versions.frontend}</Typography>

                    <Typography><strong>Agent Version:</strong></Typography>
                    <Typography>{versions.agent}</Typography>

                    <Typography><strong>API Version:</strong></Typography>
                    <Typography>{versions.api}</Typography>

                    <Typography><strong>Database:</strong></Typography>
                    <Typography>{versions.database}</Typography>
                </Box>
            </Paper>

            <Paper sx={{ p: 3 }}>
                <Typography variant="h6" gutterBottom>
                    Project Information
                </Typography>
                <Typography paragraph>
                    KrakenHashes is a distributed password cracking management system designed to coordinate and manage password cracking tasks across multiple agents.
                </Typography>
                <Typography paragraph>
                    This is an open-source project licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
                </Typography>
                <Box sx={{ mt: 2 }}>
                    <Link 
                        href="https://github.com/ZerkerEOD/krakenhashes" 
                        target="_blank" 
                        rel="noopener noreferrer"
                    >
                        GitHub Repository
                    </Link>
                </Box>
            </Paper>
        </Box>
    );
};

export default About; 