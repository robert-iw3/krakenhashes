import React, { useState, useEffect } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  Typography,
  Box,
} from '@mui/material';
import JobRow from './JobRow';
import { JobSummary, PaginationInfo } from '../../types/jobs';
import { getMaxPriorityForUsers } from '../../services/systemSettings';

interface JobsTableProps {
  jobs: JobSummary[];
  pagination?: PaginationInfo;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
  currentPage: number;
  pageSize: number;
  onJobUpdated?: () => void;
}

const JobsTable: React.FC<JobsTableProps> = ({
  jobs,
  pagination,
  onPageChange,
  onPageSizeChange,
  currentPage,
  pageSize,
  onJobUpdated,
}) => {
  const [maxPriority, setMaxPriority] = useState<number>(10); // Default to 10 as fallback

  useEffect(() => {
    // Fetch the max priority setting from the API
    getMaxPriorityForUsers()
      .then(config => {
        setMaxPriority(config.max_priority);
      })
      .catch(error => {
        console.error('Failed to fetch max priority setting:', error);
        // Keep default value of 10 on error
      });
  }, []);

  const handleChangePage = (event: unknown, newPage: number) => {
    onPageChange(newPage + 1); // Material-UI uses 0-based indexing
  };

  const handleChangeRowsPerPage = (event: React.ChangeEvent<HTMLInputElement>) => {
    const newPageSize = parseInt(event.target.value, 10);
    onPageSizeChange(newPageSize);
  };

  if (!jobs || jobs.length === 0) {
    return (
      <Box sx={{ p: 4, textAlign: 'center' }}>
        <Typography variant="h6" color="text.secondary">
          No jobs found
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Create a new job from the Hashlists page to get started.
        </Typography>
      </Box>
    );
  }

  // Separate active and completed jobs
  const activeJobs = jobs.filter(job => 
    ['pending', 'running', 'paused'].includes(job.status.toLowerCase())
  );
  const completedJobs = jobs.filter(job => 
    !['pending', 'running', 'paused'].includes(job.status.toLowerCase())
  );

  return (
    <>
      <TableContainer>
        <Table stickyHeader>
          <TableHead>
            <TableRow>
              <TableCell>Job Name</TableCell>
              <TableCell>Hashlist</TableCell>
              <TableCell>Created By</TableCell>
              <TableCell align="center">Progress</TableCell>
              <TableCell align="center">Keyspace</TableCell>
              <TableCell align="center">Cracked</TableCell>
              <TableCell align="center">Agents</TableCell>
              <TableCell align="center">Priority</TableCell>
              <TableCell align="center">Max Agents</TableCell>
              <TableCell align="center">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {/* Active Jobs */}
            {activeJobs.map((job, index) => (
              <JobRow 
                key={job.id} 
                job={job} 
                onJobUpdated={onJobUpdated}
                isLastActiveJob={index === activeJobs.length - 1 && completedJobs.length > 0}
                maxPriority={maxPriority}
              />
            ))}
            
            {/* Visual separator between active and completed jobs */}
            {activeJobs.length > 0 && completedJobs.length > 0 && (
              <TableRow>
                <TableCell colSpan={10} sx={{ py: 1, bgcolor: 'action.hover' }}>
                  <Typography variant="body2" sx={{ fontWeight: 'medium', color: 'text.secondary', textAlign: 'center' }}>
                    Completed Jobs
                  </Typography>
                </TableCell>
              </TableRow>
            )}
            
            {/* Completed Jobs */}
            {completedJobs.map((job) => (
              <JobRow 
                key={job.id} 
                job={job} 
                onJobUpdated={onJobUpdated} 
                isCompletedSection={true}
                maxPriority={maxPriority}
              />
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      
      {pagination && (
        <TablePagination
          rowsPerPageOptions={[25, 50, 100, 200]}
          component="div"
          count={pagination.total}
          rowsPerPage={pageSize}
          page={currentPage - 1} // Material-UI uses 0-based indexing
          onPageChange={handleChangePage}
          onRowsPerPageChange={handleChangeRowsPerPage}
          showFirstButton
          showLastButton
        />
      )}
    </>
  );
};

export default JobsTable;