import * as vscode from 'vscode';
import * as crypto from 'crypto';
import { ApplicationProfile } from '../profiler/project-profiler';
import { ProfileStoreData, StoredProfile } from '../models/profile-store';

/**
 * Service for managing application profiles storage
 */
export class ProfileStorageService {
  private static readonly STORE_KEY = 'secureflow.profileStore';
  private context: vscode.ExtensionContext;
  private data: ProfileStoreData;
  
  /**
   * Create a new ProfileStorageService
   * @param context Extension context
   */
  constructor(context: vscode.ExtensionContext) {
    this.context = context;
    this.data = this.loadData();
  }
  
  /**
   * Load profile store data from storage
   */
  private loadData(): ProfileStoreData {
    const data = this.context.globalState.get<ProfileStoreData>(ProfileStorageService.STORE_KEY);
    if (!data) {
      return {
        profiles: {},
        workspaceProfiles: {},
        version: 1
      };
    }
    return data;
  }
  
  /**
   * Save profile store data to storage
   */
  private async saveData(): Promise<void> {
    await this.context.globalState.update(ProfileStorageService.STORE_KEY, this.data);
  }
  
  /**
   * Generate a unique ID for a profile
   */
  private generateId(profile: ApplicationProfile, workspaceFolderUri: string): string {
    const input = `${workspaceFolderUri}:${profile.path}:${profile.category}:${profile.name}:${Date.now()}`;
    return crypto.createHash('md5').update(input).digest('hex');
  }
  
  /**
   * Store an application profile
   * 
   * @param profile The application profile to store
   * @param workspaceFolderUri The workspace folder URI containing this application
   * @param isActive Whether this profile should be set as active
   * @returns The stored profile with ID
   */
  public async storeProfile(
    profile: ApplicationProfile, 
    workspaceFolderUri: string, 
    isActive: boolean = false
  ): Promise<StoredProfile> {
    // Generate a unique ID for this profile
    const id = this.generateId(profile, workspaceFolderUri);
    
    // Create the stored profile with metadata
    const storedProfile: StoredProfile = {
      ...profile,
      id,
      timestamp: Date.now(),
      isActive,
      workspaceFolderUri
    };
    
    // Add to profiles collection
    this.data.profiles[id] = storedProfile;
    
    // Initialize workspace profiles array if needed
    if (!this.data.workspaceProfiles[workspaceFolderUri]) {
      this.data.workspaceProfiles[workspaceFolderUri] = [];
    }
    
    // Add to workspace profiles if not already present
    if (!this.data.workspaceProfiles[workspaceFolderUri].includes(id)) {
      this.data.workspaceProfiles[workspaceFolderUri].push(id);
    }
    
    // If this profile should be active, deactivate all others in this workspace
    if (isActive) {
      this.deactivateOtherProfiles(workspaceFolderUri, id);
    }
    
    // Save changes
    await this.saveData();
    
    return storedProfile;
  }
  
  /**
   * Get all profiles for a workspace folder
   * 
   * @param workspaceFolderUri The workspace folder URI
   * @returns Array of stored profiles
   */
  public getWorkspaceProfiles(workspaceFolderUri: string): StoredProfile[] {
    const profileIds = this.data.workspaceProfiles[workspaceFolderUri] || [];
    return profileIds.map(id => this.data.profiles[id]).filter(Boolean);
  }
  
  /**
   * Get all stored profiles
   * 
   * @returns Array of all stored profiles
   */
  public getAllProfiles(): StoredProfile[] {
    return Object.values(this.data.profiles);
  }
  
  /**
   * Get a profile by ID
   * 
   * @param id The profile ID
   * @returns The stored profile or undefined if not found
   */
  public getProfileById(id: string): StoredProfile | undefined {
    return this.data.profiles[id];
  }
  
  /**
   * Get the active profile for a workspace folder
   * 
   * @param workspaceFolderUri The workspace folder URI
   * @returns The active profile or undefined if none active
   */
  public getActiveProfile(workspaceFolderUri: string): StoredProfile | undefined {
    const profiles = this.getWorkspaceProfiles(workspaceFolderUri);
    return profiles.find(profile => profile.isActive);
  }
  
  /**
   * Set a profile as active for its workspace
   * 
   * @param profileId The ID of the profile to activate
   * @returns The activated profile or undefined if not found
   */
  public async activateProfile(profileId: string): Promise<StoredProfile | undefined> {
    const profile = this.data.profiles[profileId];
    if (!profile) {
      return undefined;
    }
    
    // Deactivate other profiles in this workspace
    this.deactivateOtherProfiles(profile.workspaceFolderUri, profileId);
    
    // Set this profile as active
    profile.isActive = true;
    
    // Save changes
    await this.saveData();
    
    return profile;
  }
  
  /**
   * Deactivate all other profiles in a workspace
   * 
   * @param workspaceFolderUri The workspace folder URI
   * @param exceptProfileId The profile ID to exclude from deactivation
   */
  private deactivateOtherProfiles(workspaceFolderUri: string, exceptProfileId: string): void {
    const profileIds = this.data.workspaceProfiles[workspaceFolderUri] || [];
    
    for (const profileId of profileIds) {
      if (profileId !== exceptProfileId && this.data.profiles[profileId]) {
        this.data.profiles[profileId].isActive = false;
      }
    }
  }
  
  /**
   * Delete a profile by ID
   * 
   * @param profileId The ID of the profile to delete
   * @returns True if the profile was deleted, false otherwise
   */
  public async deleteProfile(profileId: string): Promise<boolean> {
    const profile = this.data.profiles[profileId];
    if (!profile) {
      return false;
    }
    
    // Remove from workspace profiles
    const workspaceFolderUri = profile.workspaceFolderUri;
    if (this.data.workspaceProfiles[workspaceFolderUri]) {
      this.data.workspaceProfiles[workspaceFolderUri] = 
        this.data.workspaceProfiles[workspaceFolderUri].filter(id => id !== profileId);
    }
    
    // Remove from profiles collection
    delete this.data.profiles[profileId];
    
    // Save changes
    await this.saveData();
    
    return true;
  }
  
  /**
   * Clear all profiles
   */
  public async clearAllProfiles(): Promise<void> {
    this.data = {
      profiles: {},
      workspaceProfiles: {},
      version: this.data.version
    };
    
    await this.saveData();
  }
}
