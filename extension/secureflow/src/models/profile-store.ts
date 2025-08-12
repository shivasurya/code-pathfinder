import { ApplicationProfile } from '../profiler/project-profiler';

/**
 * Stored application profile with additional metadata
 */
export interface StoredProfile extends ApplicationProfile {
  /**
   * Unique identifier for the stored profile
   */
  id: string;

  /**
   * Timestamp when the profile was created/updated
   */
  timestamp: number;

  /**
   * Whether this profile is currently active
   */
  isActive: boolean;

  /**
   * The absolute path to the workspace folder containing this application
   */
  workspaceFolderUri: string;
}

/**
 * Schema for the profile store data
 */
export interface ProfileStoreData {
  /**
   * Map of profile IDs to stored profiles
   */
  profiles: { [id: string]: StoredProfile };

  /**
   * Map of workspace folder URIs to profile IDs contained in that workspace
   */
  workspaceProfiles: { [workspaceFolderUri: string]: string[] };

  /**
   * Version of the store schema, for future migrations
   */
  version: number;
}
